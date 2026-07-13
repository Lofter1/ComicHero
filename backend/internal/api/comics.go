package api

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func RegisterComicRoutes(api huma.API, db *sqlx.DB, covers *CoverCache) {
	huma.Register(api, huma.Operation{
		OperationID: "listComics",
		Tags:        []string{tagComics},
		Summary:     "List comics",
		Description: "Returns comics tracked for reading orders. Titles are generated from series metadata. Results are ordered by series, series year, then issue and can be narrowed with text, series, publisher, read-status, or reading-order filters.",
		Method:      http.MethodGet,
		Path:        "/comics",
		Errors:      errsRead,
	}, func(ctx context.Context, input *ComicListInput) (*ComicListOutput, error) {
		return listComics(ctx, db, input)
	})

	huma.Register(api, huma.Operation{
		OperationID: "getComic",
		Tags:        []string{tagComics},
		Summary:     "Get a comic",
		Description: "Returns a comic by ID, including the reading orders that contain it.",
		Method:      http.MethodGet,
		Path:        "/comics/{id}",
		Errors:      errsRead,
	}, func(ctx context.Context, input *ComicInput) (*ComicDetailOutput, error) {
		return getComic(ctx, db, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "createComic",
		Tags:          []string{tagComics},
		Summary:       "Create a comic",
		Description:   "Creates a local comic record. Series is required and the display title is generated from the submitted metadata.",
		Method:        http.MethodPost,
		Path:          "/comics",
		DefaultStatus: 201,
		Errors:        errsWrite,
	}, func(ctx context.Context, input *CreateComicInput) (*ComicDetailOutput, error) {
		return createComic(ctx, db, covers, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateComic",
		Tags:        []string{tagComics},
		Summary:     "Update a comic",
		Description: "Replaces the editable metadata and read status for a comic. Use the read-status endpoint when only toggling read state.",
		Method:      http.MethodPut,
		Path:        "/comics/{id}",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *UpdateComicInput) (*ComicDetailOutput, error) {
		return updateComic(ctx, db, covers, input.ID, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateComicReadStatus",
		Tags:        []string{tagComics},
		Summary:     "Update comic read status",
		Description: "Marks a comic as read, unread, skipped, or unskipped without updating the rest of the comic metadata.",
		Method:      http.MethodPatch,
		Path:        "/comic/{id}/read",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *UpdateComicReadInput) (*ComicDetailOutput, error) {
		return updateComicReadStatus(ctx, db, input.ID, input)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "deleteComic",
		Tags:          []string{tagComics},
		Summary:       "Delete a comic",
		Description:   "Deletes a comic by ID and removes its reading-order, arc, character, and user-progress links. Admin access is required.",
		Method:        http.MethodDelete,
		Path:          "/comics/{id}",
		DefaultStatus: 204,
		Errors:        errsWrite,
	}, func(ctx context.Context, input *ComicInput) (*struct{}, error) {
		return deleteComic(ctx, db, input.ID)
	})
}

func listComics(ctx context.Context, db *sqlx.DB, input *ComicListInput) (*ComicListOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	query, args, err := comicListQuery(input, userID)
	if err != nil {
		return nil, err
	}

	total, err := countRows(ctx, db, query, args)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to count comics")
	}

	query, args, limit, offset := paginatedQuery(query, args, input.Limit, input.Offset)

	comics := []Comic{}
	if err := db.SelectContext(ctx, &comics, query, args...); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch comics")
	}

	var pagination PaginationHeaders
	comics, pagination = pageItems(comics, limit, offset, total)
	hydrateComicTitles(comics)

	return &ComicListOutput{PaginationHeaders: pagination, Body: comics}, nil
}

func comicListQuery(input *ComicListInput, userID int) (string, []any, error) {
	query := newSelectQuery(`
		SELECT c.*,
			COALESCE(uc.read, 0) AS read, COALESCE(uc.skipped, 0) AS skipped
		FROM comics c
		LEFT JOIN user_comics uc ON uc.comic_id = c.id AND uc.user_id = ?
	`)
	query.args = append(query.args, userID)

	if input.Query != "" {
		rawQuery := strings.TrimSpace(input.Query)
		issuePattern := regexp.MustCompile(`#\s*([A-Za-z0-9.\-/]+)`)
		issueMatches := issuePattern.FindAllStringSubmatch(rawQuery, -1)

		for _, match := range issueMatches {
			query.where("c.issue = ?", match[1])
		}

		textQuery := issuePattern.ReplaceAllString(rawQuery, " ")
		terms := strings.Fields(textQuery)

		for _, term := range terms {
			search := "%" + term + "%"
			query.where(`(
				c.series LIKE ?
				OR CAST(c.series_year AS TEXT) LIKE ?
				OR CAST(c.issue AS TEXT) LIKE ?
				OR c.publisher LIKE ?
				OR c.description LIKE ?
			)`, search, search, search, search, search)
		}
	}

	if input.Series != "" {
		query.where("c.series LIKE ?", "%"+input.Series+"%")
	}

	if input.Publisher != "" {
		query.where("c.publisher LIKE ?", "%"+input.Publisher+"%")
	}

	if input.Status != "" && input.Status != "all" {
		statusSQL, err := comicStatusWhere(input.Status)
		if err != nil {
			return "", nil, err
		}
		if statusSQL != "" {
			query.where(statusSQL)
		}
	} else if read, ok, err := parseOptionalBool(input.Read, "read"); err != nil {
		return "", nil, err
	} else if ok {
		query.where("COALESCE(uc.read, 0) = ?", read)
	}
	if input.Status == "" || input.Status == "all" {
		if skipped, ok, err := parseOptionalBool(input.Skipped, "skipped"); err != nil {
			return "", nil, err
		} else if ok {
			query.where("COALESCE(uc.skipped, 0) = ?", skipped)
		}
	}

	if input.ReadingOrderID > 0 {
		query.where(`
		(
			EXISTS (
				SELECT 1 FROM reading_order_comics roc
				WHERE roc.comic_id = c.id AND roc.reading_order_id = ?
			)
			OR EXISTS (
				SELECT 1 FROM reading_order_children child
				JOIN reading_order_comics child_roc ON child_roc.reading_order_id = child.child_reading_order_id
				WHERE child.parent_reading_order_id = ? AND child_roc.comic_id = c.id
			)
		)
		`, input.ReadingOrderID, input.ReadingOrderID)
	}

	if input.ArcID > 0 {
		query.where(`
			EXISTS (
				SELECT 1 FROM arc_comics ac
				WHERE ac.comic_id = c.id AND ac.arc_id = ?
			)
		`, input.ArcID)
	}

	if input.CharacterID > 0 {
		query.where(`
			EXISTS (
				SELECT 1 FROM comic_characters cc
				WHERE cc.comic_id = c.id AND cc.character_id = ?
			)
		`, input.CharacterID)
	}

	if input.SeriesID > 0 {
		query.where("c.series_id = ?", input.SeriesID)
	}

	query.orderBy(comicListOrder(input.Sort, input.Direction))

	sql, args := query.build()
	return sql, args, nil
}

func comicStatusWhere(status string) (string, error) {
	parts := strings.Split(status, ",")
	clauses := make([]string, 0, len(parts))
	seen := map[string]bool{}

	for _, part := range parts {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" || part == "all" || seen[part] {
			continue
		}
		seen[part] = true
		switch part {
		case "unread":
			clauses = append(clauses, "(COALESCE(uc.read, 0) = 0 AND COALESCE(uc.skipped, 0) = 0)")
		case "read":
			clauses = append(clauses, "(COALESCE(uc.read, 0) = 1 AND COALESCE(uc.skipped, 0) = 0)")
		case "skipped":
			clauses = append(clauses, "COALESCE(uc.skipped, 0) = 1")
		default:
			return "", huma.Error400BadRequest("status must include only unread, read, skipped, or all")
		}
	}

	if len(clauses) == 0 || len(clauses) == 3 {
		return "", nil
	}
	return "(" + strings.Join(clauses, " OR ") + ")", nil
}

func comicListOrder(sort, direction string) string {
	dir := sortDirection(direction)
	if dir == "ASC" {
		dir = ""
	}

	spaceDir := ""
	if dir != "" {
		spaceDir = " " + dir
	}

	switch sort {
	case "title":
		return "ORDER BY c.series" + spaceDir + ", c.series_year" + spaceDir + ", CAST(c.issue AS REAL)" + spaceDir + ", c.issue" + spaceDir
	case "date":
		return "ORDER BY c.cover_date" + spaceDir + ", c.series" + spaceDir + ", c.series_year" + spaceDir + ", CAST(c.issue AS REAL)" + spaceDir + ", c.issue" + spaceDir
	case "publisher":
		return "ORDER BY c.publisher" + spaceDir + ", c.series" + spaceDir + ", c.series_year" + spaceDir + ", CAST(c.issue AS REAL)" + spaceDir + ", c.issue" + spaceDir
	case "read":
		return "ORDER BY COALESCE(uc.read, 0)" + spaceDir + ", c.series" + spaceDir + ", c.series_year" + spaceDir + ", CAST(c.issue AS REAL)" + spaceDir + ", c.issue" + spaceDir
	default:
		return "ORDER BY c.series" + spaceDir + ", c.series_year" + spaceDir + ", CAST(c.issue AS REAL)" + spaceDir + ", c.issue" + spaceDir
	}
}

func getComic(ctx context.Context, db *sqlx.DB, id int) (*ComicDetailOutput, error) {
	comic, err := getComicRow(ctx, db, id)
	if err != nil {
		return nil, err
	}

	orders := []ReadingOrder{}
	if err := db.SelectContext(ctx, &orders, `
		SELECT ro.*
		FROM reading_orders ro
		JOIN reading_order_comics roc ON roc.reading_order_id = ro.id
		WHERE roc.comic_id = ?
		ORDER BY ro.name
	`, id); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch reading orders")
	}

	arcs := []Arc{}
	if err := db.SelectContext(ctx, &arcs, `
		SELECT a.*
		FROM arcs a
		JOIN arc_comics ac ON ac.arc_id = a.id
		WHERE ac.comic_id = ?
		ORDER BY a.name
	`, id); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch arcs")
	}

	characters := []Character{}
	if err := db.SelectContext(ctx, &characters, `
		SELECT ch.*, COUNT(cc_all.comic_id) AS appearance_count
		FROM characters ch
		JOIN comic_characters cc ON cc.character_id = ch.id
		LEFT JOIN comic_characters cc_all ON cc_all.character_id = ch.id
		WHERE cc.comic_id = ?
		GROUP BY ch.id
		ORDER BY ch.name
	`, id); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch characters")
	}

	if err := hydrateCharacterAliases(ctx, db, characters); err != nil {
		return nil, err
	}

	seriesID := comic.SeriesID
	if seriesID == nil {
		var localSeriesID int
		if err := db.GetContext(ctx, &localSeriesID, `
			SELECT id FROM series WHERE name = ? AND series_year = ?
		`, comic.Series, comic.SeriesYear); err != nil {
			if err != sql.ErrNoRows {
				return nil, huma.Error500InternalServerError("failed to fetch comic series")
			}
		} else {
			seriesID = &localSeriesID
		}
	}

	return &ComicDetailOutput{
		Body: ComicDetail{
			Comic:         comic,
			SeriesID:      seriesID,
			ReadingOrders: orders,
			Arcs:          arcs,
			Characters:    characters,
		},
	}, nil
}

func getComicRow(ctx context.Context, db *sqlx.DB, id int) (Comic, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return Comic{}, err
	}
	var comic Comic
	if err := db.GetContext(ctx, &comic, `
		SELECT c.*,
			COALESCE(uc.read, 0) AS read, COALESCE(uc.skipped, 0) AS skipped
		FROM comics c
		LEFT JOIN user_comics uc ON uc.comic_id = c.id AND uc.user_id = ?
		WHERE c.id = ?
	`, userID, id); err != nil {
		if err == sql.ErrNoRows {
			return Comic{}, huma.Error404NotFound("comic not found")
		}
		return Comic{}, huma.Error500InternalServerError("failed to fetch comic")
	}

	hydrateComicTitle(&comic)
	return comic, nil
}

func createComic(ctx context.Context, db *sqlx.DB, covers *CoverCache, payload ComicPayload) (*ComicDetailOutput, error) {
	var err error
	payload.CoverImage, err = localCoverURL(ctx, covers, payload.CoverImage)
	if err != nil {
		return nil, err
	}

	seriesID, err := ensureSeriesRow(ctx, db, payload.Series, payload.SeriesYear)
	if err != nil {
		return nil, err
	}

	result, err := db.ExecContext(ctx, `
		INSERT INTO comics (series_id, series, series_year, issue, publisher, cover_date, cover_image, description)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, nullableSeriesID(seriesID),
		payload.Series,
		payload.SeriesYear,
		payload.Issue,
		payload.Publisher,
		payload.CoverDate,
		payload.CoverImage,
		payload.Description,
	)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create comic")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get new id")
	}
	if err := setComicReadStatusForCurrentUser(ctx, db, int(id), payload.Read, true, nil); err != nil {
		return nil, err
	}
	return getComic(ctx, db, int(id))
}

func updateComic(ctx context.Context, db *sqlx.DB, covers *CoverCache, id int, payload ComicPayload) (*ComicDetailOutput, error) {
	var err error
	payload.CoverImage, err = localCoverURL(ctx, covers, payload.CoverImage)
	if err != nil {
		return nil, err
	}

	seriesID, err := ensureSeriesRow(ctx, db, payload.Series, payload.SeriesYear)
	if err != nil {
		return nil, err
	}

	result, err := db.ExecContext(ctx, `
		UPDATE comics
		SET series_id = ?, series = ?, series_year = ?, issue = ?, publisher = ?, cover_date = ?, cover_image = ?, description = ?
		WHERE id = ?
	`, nullableSeriesID(seriesID),
		payload.Series,
		payload.SeriesYear,
		payload.Issue,
		payload.Publisher,
		payload.CoverDate,
		payload.CoverImage,
		payload.Description,
		id,
	)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to update comic")
	}

	if err := requireRowsAffected(result, "comic not found"); err != nil {
		return nil, err
	}

	if err := setComicReadStatusForCurrentUser(ctx, db, id, payload.Read, true, nil); err != nil {
		return nil, err
	}

	return getComic(ctx, db, id)
}

func updateComicReadStatus(ctx context.Context, db *sqlx.DB, id int, input *UpdateComicReadInput) (*ComicDetailOutput, error) {
	read := false
	readProvided := false
	var skipped *bool
	if input != nil {
		if input.Body.Read != nil {
			read = *input.Body.Read
			readProvided = true
		}
		skipped = input.Body.Skipped
	}
	if err := setComicReadStatusForCurrentUser(ctx, db, id, read, readProvided, skipped); err != nil {
		return nil, err
	}

	return getComic(ctx, db, id)
}

func setComicReadStatusForCurrentUser(ctx context.Context, db *sqlx.DB, comicID int, read bool, readProvided bool, skipped *bool) error {
	userID, err := currentUserID(ctx)
	if err != nil {
		return err
	}
	readAt := ""
	if read {
		readAt = currentTimestamp()
	}

	skippedValue := 0
	if skipped != nil {
		skippedValue = boolInt(*skipped)
	}

	result, err := db.ExecContext(ctx, `
		INSERT INTO user_comics (comic_id, user_id, read, skipped, read_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(comic_id, user_id) DO UPDATE SET
			read = CASE WHEN ? THEN excluded.read ELSE user_comics.read END,
			skipped = CASE WHEN ? THEN excluded.skipped ELSE user_comics.skipped END,
			read_at = CASE WHEN ? THEN excluded.read_at ELSE user_comics.read_at END
	`, comicID, userID, boolInt(read), skippedValue, readAt, readProvided, skipped != nil, readProvided)
	if err != nil {
		return huma.Error500InternalServerError("failed to update comic read status")
	}
	if _, err := result.RowsAffected(); err != nil {
		return huma.Error500InternalServerError("failed to check comic read status update")
	}
	return nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func deleteComic(ctx context.Context, db *sqlx.DB, id int) (*struct{}, error) {
	if _, err := requireAdminUser(ctx, db); err != nil {
		return nil, err
	}
	result, err := db.ExecContext(ctx, `DELETE FROM comics WHERE id = ?`, id)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to delete comic")
	}

	if err := requireRowsAffected(result, "comic not found"); err != nil {
		return nil, err
	}

	return &struct{}{}, nil
}

func hydrateComicTitles(comics []Comic) {
	for i := range comics {
		hydrateComicTitle(&comics[i])
	}
}

func hydrateReadingOrderComicTitles(comics []ReadingOrderComic) {
	for i := range comics {
		hydrateComicTitle(&comics[i].Comic)
	}
}

func hydrateComicTitle(comic *Comic) {
	comic.Title = comicTitle(*comic)
}

func comicTitle(comic Comic) string {
	title := comic.Series
	if title == "" {
		title = "Unknown Series"
	}

	if comic.SeriesYear > 0 {
		title = fmt.Sprintf("%s (%d)", title, comic.SeriesYear)
	}

	if comic.Series != "" || comic.Issue != "" {
		title = fmt.Sprintf("%s #%s", title, comic.Issue)
	}

	return title
}

func localCoverURL(ctx context.Context, covers *CoverCache, source string) (string, error) {
	localURL, err := covers.LocalURL(ctx, source)
	if err != nil {
		if isRemoteCoverSource(source) {
			log.Printf("cover cache: skipping remote image %q: %v", source, err)
			return "", nil
		}
		return "", huma.Error502BadGateway(err.Error())
	}

	return localURL, nil
}

func isRemoteCoverSource(source string) bool {
	source = strings.ToLower(strings.TrimSpace(source))
	return strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")
}
