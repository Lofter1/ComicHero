package api

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func RegisterReadingOrderRoutes(api huma.API, db *sqlx.DB) {
	huma.Register(api, huma.Operation{
		OperationID: "listReadingOrders",
		Tags:        []string{tagReadingOrders},
		Summary:     "List reading orders",
		Description: "Returns reading orders with computed read progress. Results can be filtered by text, favorite status, or a comic they contain.",
		Method:      http.MethodGet,
		Path:        "/readingOrders",
		Errors:      errsRead,
	}, func(ctx context.Context, input *ReadingOrderListInput) (*ReadingOrderListOutput, error) {
		return listReadingOrders(ctx, db, input)
	})

	huma.Register(api, huma.Operation{
		OperationID: "getReadingOrder",
		Tags:        []string{tagReadingOrders},
		Summary:     "Get a reading order",
		Description: "Returns a reading order by ID, including its comics in reading order and computed progress.",
		Method:      http.MethodGet,
		Path:        "/readingOrders/{id}",
		Errors:      errsRead,
	}, func(ctx context.Context, input *ReadingOrderInput) (*ReadingOrderDetailOutput, error) {
		return getReadingOrder(ctx, db, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "createReadingOrder",
		Tags:          []string{tagReadingOrders},
		Summary:       "Create a reading order",
		Description:   "Creates a reading order with a name, description, and favorite flag.",
		Method:        http.MethodPost,
		Path:          "/readingOrders",
		DefaultStatus: 201,
		Errors:        errsWrite,
	}, func(ctx context.Context, input *CreateReadingOrderInput) (*CreateReadingOrderOutput, error) {
		return createReadingOrder(ctx, db, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateReadingOrder",
		Tags:        []string{tagReadingOrders},
		Summary:     "Update a reading order",
		Description: "Updates a reading order's name, description, and favorite flag. It does not change the order's comic entries.",
		Method:      http.MethodPut,
		Path:        "/readingOrders/{id}",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *UpdateReadingOrderInput) (*ReadingOrderDetailOutput, error) {
		return updateReadingOrder(ctx, db, input.ID, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "deleteReadingOrder",
		Tags:          []string{tagReadingOrders},
		Summary:       "Delete a reading order",
		Description:   "Deletes a reading order by ID and clears its comic-entry links.",
		Method:        http.MethodDelete,
		Path:          "/readingOrders/{id}",
		DefaultStatus: 204,
		Errors:        errsWrite,
	}, func(ctx context.Context, input *ReadingOrderInput) (*struct{}, error) {
		return deleteReadingOrder(ctx, db, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "setReadingOrderComics",
		Tags:        []string{tagReadingOrders},
		Summary:     "Set reading order comics",
		Description: "Replaces every comic entry in a reading order. Entry order is the submitted array order, duplicate comic IDs are allowed, and the comics form supports per-entry comments.",
		Method:      http.MethodPut,
		Path:        "/readingOrders/{id}/comics",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *SetReadingOrderComicsInput) (*ReadingOrderDetailOutput, error) {
		return setReadingOrderComics(ctx, db, input)
	})
}

func computeProgress(comics []ReadingOrderComic) float64 {
	if len(comics) == 0 {
		return 100
	}

	read := 0
	for _, c := range comics {
		if c.Read {
			read++
		}
	}
	return float64(read) / float64(len(comics))
}

func listReadingOrders(ctx context.Context, db *sqlx.DB, input *ReadingOrderListInput) (*ReadingOrderListOutput, error) {
	query, args, err := readingOrderListQuery(input)
	if err != nil {
		return nil, err
	}
	total, err := countRows(ctx, db, query, args)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to count reading orders")
	}
	query, args, limit, offset := paginatedQuery(query, args, input.Limit, input.Offset)

	readingOrders := []ReadingOrder{}
	if err := db.SelectContext(ctx, &readingOrders, query, args...); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch reading orders")
	}
	var pagination PaginationHeaders
	readingOrders, pagination = pageItems(readingOrders, limit, offset, total)
	return &ReadingOrderListOutput{PaginationHeaders: pagination, Body: readingOrders}, nil
}

func readingOrderListQuery(input *ReadingOrderListInput) (string, []any, error) {
	query := newSelectQuery(`
		SELECT
			ro.id,
			ro.metron_reading_list_id,
			ro.name,
			ro.description,
			ro.image,
			ro.favorite,
			CASE
				WHEN COUNT(c.id) = 0 THEN 0.0
				ELSE CAST(SUM(CASE WHEN c.read = 1 THEN 1 ELSE 0 END) AS REAL) / COUNT(c.id)
			END as progress
		FROM reading_orders ro
		LEFT JOIN reading_order_comics roc ON roc.reading_order_id = ro.id
		LEFT JOIN comics c ON c.id = roc.comic_id
	`)

	if input.Query != "" {
		search := "%" + input.Query + "%"
		query.where("(ro.name LIKE ? OR ro.description LIKE ?)", search, search)
	}
	if favorite, ok, err := parseOptionalBool(input.Favorite, "favorite"); err != nil {
		return "", nil, err
	} else if ok {
		query.where("ro.favorite = ?", favorite)
	}
	if input.ComicID > 0 {
		query.where(`
			EXISTS (
				SELECT 1 FROM reading_order_comics matching_roc
				WHERE matching_roc.reading_order_id = ro.id AND matching_roc.comic_id = ?
			)
		`, input.ComicID)
	}

	query.groupBy("GROUP BY ro.id")
	query.orderBy(readingOrderListOrder(input.Sort, input.Direction))
	sql, args := query.build()
	return sql, args, nil
}

func readingOrderListOrder(sort, direction string) string {
	dir := sortDirection(direction)
	switch sort {
	case "progress":
		return "ORDER BY progress " + dir + ", ro.name " + dir
	default:
		return "ORDER BY ro.name " + dir
	}
}

func getReadingOrder(ctx context.Context, db *sqlx.DB, id int) (*ReadingOrderDetailOutput, error) {
	var readingOrder ReadingOrder
	if err := db.GetContext(ctx, &readingOrder, `
		SELECT * FROM reading_orders WHERE id = ?
	`, id); err != nil {
		return nil, huma.Error404NotFound("reading order not found")
	}

	return fetchReadingOrderDetail(ctx, db, readingOrder)
}

func fetchReadingOrderDetail(ctx context.Context, db *sqlx.DB, ro ReadingOrder) (*ReadingOrderDetailOutput, error) {
	comics, err := fetchReadingOrderComics(ctx, db, ro.ID)
	if err != nil {
		return nil, err
	}
	childOrders, err := fetchChildReadingOrders(ctx, db, ro.ID)
	if err != nil {
		return nil, err
	}
	for _, child := range childOrders {
		childComics, err := fetchReadingOrderComics(ctx, db, child.ID)
		if err != nil {
			return nil, err
		}
		for i := range childComics {
			if childComics[i].Comment == "" {
				childComics[i].Comment = "From " + child.Name
			} else {
				childComics[i].Comment = "From " + child.Name + ": " + childComics[i].Comment
			}
		}
		comics = append(comics, childComics...)
	}

	ro.Progress = computeProgress(comics)
	return &ReadingOrderDetailOutput{
		Body: ReadingOrderDetail{
			ReadingOrder:       ro,
			Comics:             comics,
			ChildReadingOrders: childOrders,
		},
	}, nil
}

func fetchReadingOrderComics(ctx context.Context, db *sqlx.DB, readingOrderID int) ([]ReadingOrderComic, error) {
	comics := []ReadingOrderComic{}
	if err := db.SelectContext(ctx, &comics, `
		SELECT c.*, roc.note AS comment, roc.tags AS tags FROM comics c
		JOIN reading_order_comics roc ON roc.comic_id = c.id
		WHERE roc.reading_order_id = ?
		ORDER BY roc.position
	`, readingOrderID); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch comics")
	}
	hydrateReadingOrderComicTitles(comics)
	return comics, nil
}

func fetchChildReadingOrders(ctx context.Context, db *sqlx.DB, readingOrderID int) ([]ReadingOrder, error) {
	orders := []ReadingOrder{}
	if err := db.SelectContext(ctx, &orders, `
		SELECT ro.* FROM reading_orders ro
		JOIN reading_order_children roc ON roc.child_reading_order_id = ro.id
		WHERE roc.parent_reading_order_id = ?
		ORDER BY roc.position, ro.name
	`, readingOrderID); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch child reading orders")
	}
	return orders, nil
}

func createReadingOrder(ctx context.Context, db *sqlx.DB, payload ReadingOrderPayload) (*CreateReadingOrderOutput, error) {
	result, err := db.ExecContext(ctx, `
		INSERT INTO reading_orders (name, description, favorite)
		VALUES (?, ?, ?)
	`, payload.Name, payload.Description, payload.Favorite)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create reading order")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get new id")
	}

	var ro ReadingOrder
	if err := db.GetContext(ctx, &ro, `
		SELECT * FROM reading_orders WHERE id = ?
	`, id); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch created reading order")
	}

	return &CreateReadingOrderOutput{Body: ro}, nil
}

func updateReadingOrder(ctx context.Context, db *sqlx.DB, id int, payload ReadingOrderPayload) (*ReadingOrderDetailOutput, error) {
	result, err := db.ExecContext(ctx, `
		UPDATE reading_orders
		SET name = ?, description = ?, favorite = ?
		WHERE id = ?
	`, payload.Name, payload.Description, payload.Favorite, id)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to update reading order")
	}
	if err := requireRowsAffected(result, "reading order not found"); err != nil {
		return nil, err
	}

	return getReadingOrder(ctx, db, id)
}

func deleteReadingOrder(ctx context.Context, db *sqlx.DB, id int) (*struct{}, error) {
	result, err := db.ExecContext(ctx, `
		DELETE FROM reading_orders WHERE id = ?
	`, id)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to delete reading order")
	}
	if err := requireRowsAffected(result, "reading order not found"); err != nil {
		return nil, err
	}

	return &struct{}{}, nil
}

func setReadingOrderComics(ctx context.Context, db *sqlx.DB, input *SetReadingOrderComicsInput) (*ReadingOrderDetailOutput, error) {
	var ro ReadingOrder
	if err := db.GetContext(ctx, &ro, `
		SELECT * FROM reading_orders WHERE id = ?
	`, input.ID); err != nil {
		return nil, huma.Error404NotFound("reading order not found")
	}

	comics := readingOrderComicItems(input)
	if err := validateReadingOrderComicIDs(ctx, db, readingOrderComicIDs(comics)); err != nil {
		return nil, err
	}
	if err := validateChildReadingOrderIDs(ctx, db, input.ID, input.Body.ReadingOrderIDs); err != nil {
		return nil, err
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to start transaction")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM reading_order_comics WHERE reading_order_id = ?
	`, input.ID); err != nil {
		return nil, huma.Error500InternalServerError("failed to clear comics")
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM reading_order_children WHERE parent_reading_order_id = ?
	`, input.ID); err != nil {
		return nil, huma.Error500InternalServerError("failed to clear child reading orders")
	}

	for i, comic := range comics {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO reading_order_comics (reading_order_id, comic_id, position, note, tags)
			VALUES (?, ?, ?, ?, ?)
		`, input.ID, comic.ComicID, i+1, comic.Comment, comic.Tags); err != nil {
			return nil, huma.Error500InternalServerError("failed to insert comic")
		}
	}
	for i, childID := range uniqueReadingOrderIDs(input.Body.ReadingOrderIDs) {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO reading_order_children (parent_reading_order_id, child_reading_order_id, position)
			VALUES (?, ?, ?)
		`, input.ID, childID, i+1); err != nil {
			return nil, huma.Error500InternalServerError("failed to insert child reading order")
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, huma.Error500InternalServerError("failed to commit")
	}

	return fetchReadingOrderDetail(ctx, db, ro)
}

func readingOrderComicItems(input *SetReadingOrderComicsInput) []ReadingOrderComicPayload {
	if len(input.Body.Comics) > 0 {
		return input.Body.Comics
	}

	comics := make([]ReadingOrderComicPayload, 0, len(input.Body.ComicIDs))
	for _, comicID := range input.Body.ComicIDs {
		comics = append(comics, ReadingOrderComicPayload{ComicID: comicID})
	}
	return comics
}

func uniqueReadingOrderIDs(ids []int) []int {
	seen := map[int]bool{}
	unique := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}
	return unique
}

func readingOrderComicIDs(comics []ReadingOrderComicPayload) []int {
	comicIDs := make([]int, 0, len(comics))
	for _, comic := range comics {
		comicIDs = append(comicIDs, comic.ComicID)
	}
	return comicIDs
}

func validateReadingOrderComicIDs(ctx context.Context, db *sqlx.DB, comicIDs []int) error {
	if len(comicIDs) == 0 {
		return nil
	}

	seen := make(map[int]struct{}, len(comicIDs))
	for _, comicID := range comicIDs {
		seen[comicID] = struct{}{}
	}

	uniqueComicIDs := make([]int, 0, len(seen))
	for comicID := range seen {
		uniqueComicIDs = append(uniqueComicIDs, comicID)
	}

	query, args, err := sqlx.In("SELECT id FROM comics WHERE id IN (?)", uniqueComicIDs)
	if err != nil {
		return huma.Error500InternalServerError("failed to validate comics")
	}
	query = db.Rebind(query)

	foundIDs := []int{}
	if err := db.SelectContext(ctx, &foundIDs, query, args...); err != nil {
		return huma.Error500InternalServerError("failed to validate comics")
	}

	for _, id := range foundIDs {
		delete(seen, id)
	}
	if len(seen) == 0 {
		return nil
	}

	missingIDs := make([]int, 0, len(seen))
	for comicID := range seen {
		missingIDs = append(missingIDs, comicID)
	}
	sort.Ints(missingIDs)

	missingIDStrings := make([]string, 0, len(missingIDs))
	for _, comicID := range missingIDs {
		missingIDStrings = append(missingIDStrings, fmt.Sprintf("%d", comicID))
	}
	return huma.Error400BadRequest(fmt.Sprintf("comic(s) not found: %s", strings.Join(missingIDStrings, ", ")))
}

func validateChildReadingOrderIDs(ctx context.Context, db *sqlx.DB, parentID int, childIDs []int) error {
	if len(childIDs) == 0 {
		return nil
	}

	seen := make(map[int]struct{}, len(childIDs))
	for _, childID := range childIDs {
		if childID == parentID {
			return huma.Error400BadRequest("reading order cannot reference itself")
		}
		seen[childID] = struct{}{}
	}

	uniqueChildIDs := make([]int, 0, len(seen))
	for childID := range seen {
		uniqueChildIDs = append(uniqueChildIDs, childID)
	}

	query, args, err := sqlx.In("SELECT id FROM reading_orders WHERE id IN (?)", uniqueChildIDs)
	if err != nil {
		return huma.Error500InternalServerError("failed to validate child reading orders")
	}
	query = db.Rebind(query)

	foundIDs := []int{}
	if err := db.SelectContext(ctx, &foundIDs, query, args...); err != nil {
		return huma.Error500InternalServerError("failed to validate child reading orders")
	}

	for _, id := range foundIDs {
		delete(seen, id)
	}
	if len(seen) == 0 {
		return nil
	}

	missingIDs := make([]int, 0, len(seen))
	for childID := range seen {
		missingIDs = append(missingIDs, childID)
	}
	sort.Ints(missingIDs)

	missingIDStrings := make([]string, 0, len(missingIDs))
	for _, childID := range missingIDs {
		missingIDStrings = append(missingIDStrings, fmt.Sprintf("%d", childID))
	}
	return huma.Error400BadRequest(fmt.Sprintf("reading order(s) not found: %s", strings.Join(missingIDStrings, ", ")))
}
