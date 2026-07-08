package api

import (
	"context"
	"database/sql"
	"fmt"
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
		Description: "Marks a comic as read or unread without updating the rest of the comic metadata.",
		Method:      http.MethodPatch,
		Path:        "/comic/{id}/read",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *UpdateComicReadInput) (*ComicDetailOutput, error) {
		return updateComicReadStatus(ctx, db, input.ID, input.Body.Read)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "deleteComic",
		Tags:          []string{tagComics},
		Summary:       "Delete a comic",
		Description:   "Deletes a comic by ID and removes it from reading orders.",
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
			COALESCE(uc.read, 0) AS read
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

	if read, ok, err := parseOptionalBool(input.Read, "read"); err != nil {
		return "", nil, err
	} else if ok {
		query.where("COALESCE(uc.read, 0) = ?", read)
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
		query.where(`
			EXISTS (
				SELECT 1 FROM series s
				WHERE s.id = ? AND s.name = c.series AND s.series_year = c.series_year
			)
		`, input.SeriesID)
	}

	query.orderBy(comicListOrder(input.Sort, input.Direction))

	sql, args := query.build()
	return sql, args, nil
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

	var seriesID *int
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
			COALESCE(uc.read, 0) AS read
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

	result, err := db.ExecContext(ctx, `
		INSERT INTO comics (series, series_year, issue, publisher, cover_date, cover_image, description)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, payload.Series,
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
	if err := setComicReadStatusForCurrentUser(ctx, db, int(id), payload.Read); err != nil {
		return nil, err
	}
	if err := ensureSeriesRow(ctx, db, payload.Series, payload.SeriesYear); err != nil {
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

	result, err := db.ExecContext(ctx, `
		UPDATE comics
		SET series = ?, series_year = ?, issue = ?, publisher = ?, cover_date = ?, cover_image = ?, description = ?
		WHERE id = ?
	`, payload.Series,
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

	if err := ensureSeriesRow(ctx, db, payload.Series, payload.SeriesYear); err != nil {
		return nil, err
	}
	if err := setComicReadStatusForCurrentUser(ctx, db, id, payload.Read); err != nil {
		return nil, err
	}

	return getComic(ctx, db, id)
}

func updateComicReadStatus(ctx context.Context, db *sqlx.DB, id int, read bool) (*ComicDetailOutput, error) {
	if err := setComicReadStatusForCurrentUser(ctx, db, id, read); err != nil {
		return nil, err
	}

	return getComic(ctx, db, id)
}

func setComicReadStatusForCurrentUser(ctx context.Context, db *sqlx.DB, comicID int, read bool) error {
	userID, err := currentUserID(ctx)
	if err != nil {
		return err
	}

	result, err := db.ExecContext(ctx, `
		INSERT INTO user_comics (comic_id, user_id, read)
		VALUES (?, ?, ?)
		ON CONFLICT(comic_id, user_id) DO UPDATE SET read = excluded.read
	`, comicID, userID, boolInt(read))
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
		return "", huma.Error502BadGateway(err.Error())
	}

	return localURL, nil
}
