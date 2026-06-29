package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

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
		Description:   "Creates a local comic record. Series is required, issue and series year must be zero or greater, and the display title is generated from the submitted metadata.",
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
	query, args, err := comicListQuery(input)
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

func comicListQuery(input *ComicListInput) (string, []any, error) {
	query := newSelectQuery("SELECT c.* FROM comics c")

	if input.Query != "" {
		search := "%" + input.Query + "%"
		query.where("(c.series LIKE ? OR CAST(c.series_year AS TEXT) LIKE ? OR CAST(c.issue AS TEXT) LIKE ? OR c.publisher LIKE ? OR c.description LIKE ?)", search, search, search, search, search)
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
		query.where("c.read = ?", read)
	}
	if input.ReadingOrderID > 0 {
		query.where(`
			EXISTS (
				SELECT 1 FROM reading_order_comics roc
				WHERE roc.comic_id = c.id AND roc.reading_order_id = ?
			)
		`, input.ReadingOrderID)
	}

	query.orderBy("ORDER BY c.series, c.series_year, c.issue")
	sql, args := query.build()
	return sql, args, nil
}

func getComic(ctx context.Context, db *sqlx.DB, id int) (*ComicDetailOutput, error) {
	comic, err := getComicRow(ctx, db, id)
	if err != nil {
		return nil, err
	}

	orders := []ReadingOrder{}
	if err := db.SelectContext(ctx, &orders, `
		SELECT ro.* FROM reading_orders ro
		JOIN reading_order_comics roc ON roc.reading_order_id = ro.id
		WHERE roc.comic_id = ?
		ORDER BY ro.name
	`, id); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch reading orders")
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

	return &ComicDetailOutput{
		Body: ComicDetail{
			Comic:         comic,
			ReadingOrders: orders,
			Characters:    characters,
		},
	}, nil
}

func getComicRow(ctx context.Context, db *sqlx.DB, id int) (Comic, error) {
	var comic Comic
	if err := db.GetContext(ctx, &comic, `
		SELECT * FROM comics WHERE id = ?
	`, id); err != nil {
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
		INSERT INTO comics (series, series_year, issue, publisher, cover_date, cover_image, description, read)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, payload.Series,
		payload.SeriesYear,
		payload.Issue,
		payload.Publisher,
		payload.CoverDate,
		payload.CoverImage,
		payload.Description,
		payload.Read,
	)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create comic")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get new id")
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
		SET series = ?, series_year = ?, issue = ?, publisher = ?, cover_date = ?, cover_image = ?, description = ?, read = ?
		WHERE id = ?
	`, payload.Series,
		payload.SeriesYear,
		payload.Issue,
		payload.Publisher,
		payload.CoverDate,
		payload.CoverImage,
		payload.Description,
		payload.Read,
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

	return getComic(ctx, db, id)
}

func updateComicReadStatus(ctx context.Context, db *sqlx.DB, id int, read bool) (*ComicDetailOutput, error) {
	result, err := db.ExecContext(ctx, `
		UPDATE comics
		SET read = ?
		WHERE id = ?
	`, read, id)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to update comic read status")
	}
	if err := requireRowsAffected(result, "comic not found"); err != nil {
		return nil, err
	}

	return getComic(ctx, db, id)
}

func deleteComic(ctx context.Context, db *sqlx.DB, id int) (*struct{}, error) {
	result, err := db.ExecContext(ctx, `
		DELETE FROM comics WHERE id = ?
	`, id)
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
	if comic.Series != "" || comic.Issue > 0 {
		title = fmt.Sprintf("%s #%d", title, comic.Issue)
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
