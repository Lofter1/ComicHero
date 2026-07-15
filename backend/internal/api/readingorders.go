package api

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func RegisterReadingOrderRoutes(api huma.API, db *sqlx.DB, covers *CoverCache) {
	huma.Register(api, huma.Operation{
		OperationID: "listReadingOrders",
		Tags:        []string{tagReadingOrders},
		Summary:     "List reading orders",
		Description: "Returns public reading orders plus private reading orders owned by the current user, with computed read progress.",
		Method:      http.MethodGet,
		Path:        "/readingOrders",
		Errors:      errsRead,
	}, func(ctx context.Context, input *ReadingOrderListInput) (*ReadingOrderListOutput, error) {
		return listReadingOrders(ctx, db, input)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "importReadingOrderCBL",
		Tags:          []string{tagReadingOrders},
		Summary:       "Import a CBL reading order",
		Description:   "Creates a reading order from CBL XML by matching CBL book entries to local comics by series, issue number, and volume or year.",
		Method:        http.MethodPost,
		Path:          "/readingOrders/cbl/import",
		DefaultStatus: 201,
		Errors:        errsWrite,
	}, func(ctx context.Context, input *ReadingOrderCBLImportInput) (*ReadingOrderCBLImportOutput, error) {
		return importReadingOrderCBL(ctx, db, input)
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
		OperationID: "exportReadingOrderCBL",
		Tags:        []string{tagReadingOrders},
		Summary:     "Export a reading order as CBL",
		Description: "Returns CBL XML for a reading order. Nested reading orders are flattened into their expanded comic issue order.",
		Method:      http.MethodGet,
		Path:        "/readingOrders/{id}/cbl",
		Errors:      errsRead,
	}, func(ctx context.Context, input *ReadingOrderInput) (*ReadingOrderCBLExportOutput, error) {
		return exportReadingOrderCBL(ctx, db, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "createReadingOrder",
		Tags:          []string{tagReadingOrders},
		Summary:       "Create a reading order",
		Description:   "Creates a public or private reading order owned by the current user.",
		Method:        http.MethodPost,
		Path:          "/readingOrders",
		DefaultStatus: 201,
		Errors:        errsWrite,
	}, func(ctx context.Context, input *CreateReadingOrderInput) (*CreateReadingOrderOutput, error) {
		return createReadingOrder(ctx, db, covers, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateReadingOrder",
		Tags:        []string{tagReadingOrders},
		Summary:     "Update a reading order",
		Description: "Updates a reading order's name, description, visibility, and favorite flag. It does not change the order's comic entries.",
		Method:      http.MethodPut,
		Path:        "/readingOrders/{id}",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *UpdateReadingOrderInput) (*ReadingOrderDetailOutput, error) {
		return updateReadingOrder(ctx, db, covers, input.ID, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "rateReadingOrder",
		Tags:        []string{tagReadingOrders},
		Summary:     "Rate a reading order",
		Description: "Sets or clears the current user's rating for a reading order. Use rating 0 to clear it.",
		Method:      http.MethodPatch,
		Path:        "/readingOrders/{id}/rating",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *UpdateReadingOrderRatingInput) (*ReadingOrderDetailOutput, error) {
		return rateReadingOrder(ctx, db, input.ID, input.Body.Rating)
	})

	huma.Register(api, huma.Operation{
		OperationID: "startReadingOrder",
		Tags:        []string{tagReadingOrders},
		Summary:     "Start a reading order",
		Description: "Formally marks a reading order as started by the current user. Repeated requests preserve the original start time.",
		Method:      http.MethodPost,
		Path:        "/readingOrders/{id}/start",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *ReadingOrderInput) (*ReadingOrderDetailOutput, error) {
		return startReadingOrder(ctx, db, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "stopReadingOrder",
		Tags:        []string{tagReadingOrders},
		Summary:     "Stop reading a reading order",
		Description: "Removes the current user's active reading-order start state without changing comic read history.",
		Method:      http.MethodDelete,
		Path:        "/readingOrders/{id}/start",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *ReadingOrderInput) (*ReadingOrderDetailOutput, error) {
		return stopReadingOrder(ctx, db, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "copyReadingOrder",
		Tags:          []string{tagReadingOrders},
		Summary:       "Copy a reading order",
		Description:   "Creates a new reading order owned by the current user by copying the source order metadata and ordered entries.",
		Method:        http.MethodPost,
		Path:          "/readingOrders/{id}/copy",
		DefaultStatus: 201,
		Errors:        errsWrite,
	}, func(ctx context.Context, input *CopyReadingOrderInput) (*ReadingOrderDetailOutput, error) {
		return copyReadingOrder(ctx, db, input.ID)
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

func readingOrderIsPublic(value *bool) bool {
	return value == nil || *value
}

func listReadingOrders(ctx context.Context, db *sqlx.DB, input *ReadingOrderListInput) (*ReadingOrderListOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	editUserID := userID
	if currentUserIsPublic(ctx) {
		editUserID = 0
	}
	query, args, err := readingOrderListQuery(input, userID, editUserID)
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

func readingOrderListQuery(input *ReadingOrderListInput, userID int, editUserID int) (string, []any, error) {
	query := newSelectQuery(`
		SELECT
			ro.id,
			ro.metron_reading_list_id,
			ro.author_user_id,
			ro.name,
			ro.description,
			ro.image,
			ro.is_public,
			COALESCE(preference.favorite, 0) AS favorite,
			(SELECT COUNT(*) FROM user_reading_orders stats WHERE stats.reading_order_id = ro.id AND stats.favorite = 1) AS favorite_count,
			(SELECT COUNT(*) FROM user_reading_orders stats WHERE stats.reading_order_id = ro.id AND stats.started_at IS NOT NULL) AS started_count,
			COALESCE(rating_summary.rating, 0) AS rating,
			COALESCE(rating_summary.rating_count, 0) AS rating_count,
			my_rating.rating AS my_rating,
			preference.started_at AS started_at,
			COALESCE(author.name, '') AS author_name,
			CASE
				WHEN ro.author_user_id = ?
					OR EXISTS (SELECT 1 FROM users current_user WHERE current_user.id = ? AND current_user.is_admin = 1)
				THEN 1 ELSE 0
			END AS can_edit,
			CASE
				WHEN COUNT(c.id) = 0 THEN 0.0
				ELSE CAST(SUM(CASE WHEN COALESCE(uc.read, 0) = 1 THEN 1 ELSE 0 END) AS REAL) / COUNT(c.id)
			END as progress
		FROM reading_orders ro
		LEFT JOIN users author ON author.id = ro.author_user_id
		LEFT JOIN reading_order_comics roc ON roc.reading_order_id = ro.id
		LEFT JOIN comics c ON c.id = roc.comic_id
		LEFT JOIN user_comics uc ON uc.comic_id = c.id AND uc.user_id = ?
		LEFT JOIN (
			SELECT reading_order_id, AVG(rating) AS rating, COUNT(*) AS rating_count
			FROM reading_order_ratings
			GROUP BY reading_order_id
		) rating_summary ON rating_summary.reading_order_id = ro.id
		LEFT JOIN reading_order_ratings my_rating
			ON my_rating.reading_order_id = ro.id AND my_rating.user_id = ?
		LEFT JOIN user_reading_orders preference
			ON preference.reading_order_id = ro.id AND preference.user_id = ?
	`)
	query.args = append(query.args, editUserID, editUserID, userID, userID, userID)
	query.where(`(
		ro.is_public = 1
		OR ro.author_user_id = ?
		OR EXISTS (SELECT 1 FROM users visibility_user WHERE visibility_user.id = ? AND visibility_user.is_admin = 1)
	)`, editUserID, editUserID)

	if input.Query != "" {
		search := "%" + input.Query + "%"
		query.where("(ro.name LIKE ? OR ro.description LIKE ?)", search, search)
	}
	if favorite, ok, err := parseOptionalBool(input.Favorite, "favorite"); err != nil {
		return "", nil, err
	} else if ok {
		query.where("COALESCE(preference.favorite, 0) = ?", favorite)
	}
	if started, ok, err := parseOptionalBool(input.Started, "started"); err != nil {
		return "", nil, err
	} else if ok && started {
		query.where("preference.started_at IS NOT NULL")
	} else if ok {
		query.where("preference.started_at IS NULL")
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
	case "rating":
		return "ORDER BY rating " + dir + ", ro.name " + dir
	case "progress":
		return "ORDER BY progress " + dir + ", ro.name " + dir
	case "favoriteCount":
		return "ORDER BY favorite_count " + dir + ", ro.name " + dir
	case "startedCount":
		return "ORDER BY started_count " + dir + ", ro.name " + dir
	default:
		return "ORDER BY ro.name " + dir
	}
}

func uploadedReadingOrderCover(covers *CoverCache, payload ReadingOrderPayload) (string, error) {
	source := strings.TrimSpace(payload.CoverImageData)
	if source == "" {
		return "", nil
	}
	image, err := decodeImageDataURL(source)
	if err != nil {
		return "", err
	}
	localURL, err := covers.StoreImage(image)
	if err != nil {
		return "", huma.Error400BadRequest(err.Error())
	}
	return localURL, nil
}

func decodeImageDataURL(source string) ([]byte, error) {
	encoded := strings.TrimSpace(source)
	if strings.HasPrefix(encoded, "data:") {
		header, body, ok := strings.Cut(encoded, ",")
		if !ok {
			return nil, huma.Error400BadRequest("cover image data URL is invalid")
		}
		if !strings.Contains(header, ";base64") {
			return nil, huma.Error400BadRequest("cover image data URL must be base64 encoded")
		}
		if !strings.HasPrefix(header, "data:image/") {
			return nil, huma.Error400BadRequest("cover image must be an image file")
		}
		encoded = body
	}

	image, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, huma.Error400BadRequest("cover image data is invalid")
	}
	return image, nil
}

func deleteUnusedCoverImage(ctx context.Context, db *sqlx.DB, covers *CoverCache, image string) error {
	image = strings.TrimSpace(image)
	if image == "" || covers == nil {
		return nil
	}

	var references int
	if err := db.GetContext(ctx, &references, `
		SELECT
			(SELECT COUNT(*) FROM reading_orders WHERE image = ?) +
			(SELECT COUNT(*) FROM comics WHERE cover_image = ?) +
			(SELECT COUNT(*) FROM characters WHERE image = ?)
	`, image, image, image); err != nil {
		return huma.Error500InternalServerError("failed to check cover image usage")
	}
	if references > 0 {
		return nil
	}
	if err := covers.RemoveLocalURL(image); err != nil {
		return huma.Error500InternalServerError("failed to delete old cover image")
	}
	return nil
}

func getReadingOrder(ctx context.Context, db *sqlx.DB, id int) (*ReadingOrderDetailOutput, error) {
	readingOrder, err := getReadingOrderRow(ctx, db, id)
	if err != nil {
		return nil, err
	}

	return fetchReadingOrderDetail(ctx, db, readingOrder)
}

func getReadingOrderRow(ctx context.Context, db *sqlx.DB, id int) (ReadingOrder, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return ReadingOrder{}, err
	}
	editUserID := userID
	if currentUserIsPublic(ctx) {
		editUserID = 0
	}
	var readingOrder ReadingOrder
	if err := db.GetContext(ctx, &readingOrder, `
		SELECT
			ro.id,
			ro.metron_reading_list_id,
			ro.author_user_id,
			ro.name,
			ro.description,
			ro.image,
			ro.is_public,
			COALESCE(preference.favorite, 0) AS favorite,
			(SELECT COUNT(*) FROM user_reading_orders stats WHERE stats.reading_order_id = ro.id AND stats.favorite = 1) AS favorite_count,
			(SELECT COUNT(*) FROM user_reading_orders stats WHERE stats.reading_order_id = ro.id AND stats.started_at IS NOT NULL) AS started_count,
			COALESCE(rating_summary.rating, 0) AS rating,
			COALESCE(rating_summary.rating_count, 0) AS rating_count,
			my_rating.rating AS my_rating,
			preference.started_at AS started_at,
			0.0 AS progress,
			COALESCE(author.name, '') AS author_name,
			CASE
				WHEN ro.author_user_id = ?
					OR EXISTS (SELECT 1 FROM users current_user WHERE current_user.id = ? AND current_user.is_admin = 1)
				THEN 1 ELSE 0
			END AS can_edit
		FROM reading_orders ro
		LEFT JOIN users author ON author.id = ro.author_user_id
		LEFT JOIN (
			SELECT reading_order_id, AVG(rating) AS rating, COUNT(*) AS rating_count
			FROM reading_order_ratings
			GROUP BY reading_order_id
		) rating_summary ON rating_summary.reading_order_id = ro.id
		LEFT JOIN reading_order_ratings my_rating
			ON my_rating.reading_order_id = ro.id AND my_rating.user_id = ?
		LEFT JOIN user_reading_orders preference
			ON preference.reading_order_id = ro.id AND preference.user_id = ?
		WHERE ro.id = ? AND (
			ro.is_public = 1
			OR ro.author_user_id = ?
			OR EXISTS (SELECT 1 FROM users visibility_user WHERE visibility_user.id = ? AND visibility_user.is_admin = 1)
		)
	`, editUserID, editUserID, userID, userID, id, editUserID, editUserID); err != nil {
		if err == sql.ErrNoRows {
			return ReadingOrder{}, huma.Error404NotFound("reading order not found")
		}
		return ReadingOrder{}, huma.Error500InternalServerError("failed to fetch reading order")
	}
	return readingOrder, nil
}

func startReadingOrder(ctx context.Context, db *sqlx.DB, id int) (*ReadingOrderDetailOutput, error) {
	if _, err := getReadingOrderRow(ctx, db, id); err != nil {
		return nil, err
	}
	if err := setContentStarted(ctx, db, "user_reading_orders", "reading_order_id", "reading_orders", id, true); err != nil {
		return nil, err
	}
	return getReadingOrder(ctx, db, id)
}

func stopReadingOrder(ctx context.Context, db *sqlx.DB, id int) (*ReadingOrderDetailOutput, error) {
	if _, err := getReadingOrderRow(ctx, db, id); err != nil {
		return nil, err
	}
	if err := setContentStarted(ctx, db, "user_reading_orders", "reading_order_id", "reading_orders", id, false); err != nil {
		return nil, err
	}
	return getReadingOrder(ctx, db, id)
}

func fetchReadingOrderDetail(ctx context.Context, db *sqlx.DB, ro ReadingOrder) (*ReadingOrderDetailOutput, error) {
	entries, err := fetchReadingOrderEntries(ctx, db, ro.ID)
	if err != nil {
		return nil, err
	}
	comics := []ReadingOrderComic{}
	childOrders := []ReadingOrder{}
	for _, entry := range entries {
		if entry.Type == "comic" && entry.Comic != nil {
			comics = append(comics, *entry.Comic)
			continue
		}
		if entry.Type != "readingOrder" || entry.ReadingOrder == nil {
			continue
		}

		child := *entry.ReadingOrder
		childOrders = append(childOrders, child)
		childComics, err := fetchReadingOrderComics(ctx, db, child.ID)
		if err != nil {
			return nil, err
		}
		source := "From " + child.Name
		if entry.Comment != "" {
			source += ": " + entry.Comment
		}
		for i := range childComics {
			if childComics[i].Comment == "" {
				childComics[i].Comment = source
			} else {
				childComics[i].Comment = source + " - " + childComics[i].Comment
			}
		}
		comics = append(comics, childComics...)
	}

	ro.Progress = computeProgress(comics)
	return &ReadingOrderDetailOutput{
		Body: ReadingOrderDetail{
			ReadingOrder:       ro,
			Entries:            entries,
			Comics:             comics,
			ChildReadingOrders: childOrders,
		},
	}, nil
}

func fetchReadingOrderEntries(ctx context.Context, db *sqlx.DB, readingOrderID int) ([]ReadingOrderEntry, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	visibilityUserID := userID
	if currentUserIsPublic(ctx) {
		visibilityUserID = 0
	}
	comics := []struct {
		ReadingOrderComic
		Position int `db:"position"`
	}{}
	if err := db.SelectContext(ctx, &comics, `
		SELECT c.*, COALESCE(uc.read, 0) AS read, COALESCE(uc.skipped, 0) AS skipped, roc.note AS comment, roc.tags AS tags, roc.position AS position FROM comics c
		JOIN reading_order_comics roc ON roc.comic_id = c.id
		LEFT JOIN user_comics uc ON uc.comic_id = c.id AND uc.user_id = ?
		WHERE roc.reading_order_id = ?
		ORDER BY roc.position
	`, userID, readingOrderID); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch comics")
	}
	for i := range comics {
		hydrateComicTitle(&comics[i].ReadingOrderComic.Comic)
	}

	children := []struct {
		ReadingOrder
		Position int    `db:"position"`
		Comment  string `db:"comment"`
	}{}
	if err := db.SelectContext(ctx, &children, `
		SELECT
			ro.id,
			ro.metron_reading_list_id,
			ro.author_user_id,
			ro.name,
			ro.description,
			ro.image,
			ro.is_public,
			COALESCE(preference.favorite, 0) AS favorite,
			0.0 AS progress,
			COALESCE(author.name, '') AS author_name,
			CASE
				WHEN ro.author_user_id = ?
					OR EXISTS (SELECT 1 FROM users current_user WHERE current_user.id = ? AND current_user.is_admin = 1)
				THEN 1 ELSE 0
			END AS can_edit,
			roc.position AS position,
			roc.note AS comment
		FROM reading_orders ro
		LEFT JOIN users author ON author.id = ro.author_user_id
		LEFT JOIN user_reading_orders preference
			ON preference.reading_order_id = ro.id AND preference.user_id = ?
		JOIN reading_order_children roc ON roc.child_reading_order_id = ro.id
		WHERE roc.parent_reading_order_id = ? AND (
			ro.is_public = 1
			OR ro.author_user_id = ?
			OR EXISTS (SELECT 1 FROM users visibility_user WHERE visibility_user.id = ? AND visibility_user.is_admin = 1)
		)
		ORDER BY roc.position, ro.name
	`, visibilityUserID, visibilityUserID, userID, readingOrderID, visibilityUserID, visibilityUserID); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch child reading orders")
	}

	positioned := make([]struct {
		position int
		entry    ReadingOrderEntry
	}, 0, len(comics)+len(children))
	for i := range comics {
		comic := comics[i].ReadingOrderComic
		positioned = append(positioned, struct {
			position int
			entry    ReadingOrderEntry
		}{
			position: comics[i].Position,
			entry: ReadingOrderEntry{
				Type:  "comic",
				Comic: &comic,
			},
		})
	}
	for i := range children {
		child := children[i].ReadingOrder
		positioned = append(positioned, struct {
			position int
			entry    ReadingOrderEntry
		}{
			position: children[i].Position,
			entry: ReadingOrderEntry{
				Type:         "readingOrder",
				ReadingOrder: &child,
				Comment:      children[i].Comment,
			},
		})
	}
	sort.SliceStable(positioned, func(i, j int) bool {
		return positioned[i].position < positioned[j].position
	})

	entries := make([]ReadingOrderEntry, 0, len(positioned))
	for _, item := range positioned {
		entries = append(entries, item.entry)
	}
	return entries, nil
}

func fetchReadingOrderComics(ctx context.Context, db *sqlx.DB, readingOrderID int) ([]ReadingOrderComic, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	comics := []ReadingOrderComic{}
	if err := db.SelectContext(ctx, &comics, `
		SELECT c.*, COALESCE(uc.read, 0) AS read, COALESCE(uc.skipped, 0) AS skipped, roc.note AS comment, roc.tags AS tags FROM comics c
		JOIN reading_order_comics roc ON roc.comic_id = c.id
		LEFT JOIN user_comics uc ON uc.comic_id = c.id AND uc.user_id = ?
		WHERE roc.reading_order_id = ?
		ORDER BY roc.position
	`, userID, readingOrderID); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch comics")
	}
	hydrateReadingOrderComicTitles(comics)
	return comics, nil
}

func createReadingOrder(ctx context.Context, db *sqlx.DB, covers *CoverCache, payload ReadingOrderPayload) (*CreateReadingOrderOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	image, err := uploadedReadingOrderCover(covers, payload)
	if err != nil {
		return nil, err
	}
	result, err := db.ExecContext(ctx, `
		INSERT INTO reading_orders (name, description, image, is_public, author_user_id)
		VALUES (?, ?, ?, ?, ?)
	`, payload.Name, payload.Description, image, readingOrderIsPublic(payload.IsPublic), userID)
	if err != nil {
		_ = deleteUnusedCoverImage(ctx, db, covers, image)
		return nil, huma.Error500InternalServerError("failed to create reading order")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get new id")
	}
	if err := setContentFavorite(ctx, db, "user_reading_orders", "reading_order_id", "reading_orders", int(id), payload.Favorite); err != nil {
		return nil, err
	}

	ro, err := getReadingOrderRow(ctx, db, int(id))
	if err != nil {
		return nil, err
	}

	return &CreateReadingOrderOutput{Body: ro}, nil
}

func updateReadingOrder(ctx context.Context, db *sqlx.DB, covers *CoverCache, id int, payload ReadingOrderPayload) (*ReadingOrderDetailOutput, error) {
	if err := requireReadingOrderEditor(ctx, db, id); err != nil {
		return nil, err
	}

	existing, err := getReadingOrderRow(ctx, db, id)
	if err != nil {
		return nil, err
	}
	image := existing.Image
	isPublic := existing.IsPublic
	if payload.IsPublic != nil {
		isPublic = *payload.IsPublic
	}
	if strings.TrimSpace(payload.CoverImageData) != "" {
		image, err = uploadedReadingOrderCover(covers, payload)
		if err != nil {
			return nil, err
		}
	}

	result, err := db.ExecContext(ctx, `
		UPDATE reading_orders
		SET name = ?, description = ?, image = ?, is_public = ?
		WHERE id = ?
	`, payload.Name, payload.Description, image, isPublic, id)
	if err != nil {
		if image != existing.Image {
			_ = deleteUnusedCoverImage(ctx, db, covers, image)
		}
		return nil, huma.Error500InternalServerError("failed to update reading order")
	}
	if err := requireRowsAffected(result, "reading order not found"); err != nil {
		return nil, err
	}
	if err := setContentFavorite(ctx, db, "user_reading_orders", "reading_order_id", "reading_orders", id, payload.Favorite); err != nil {
		return nil, err
	}

	detail, err := getReadingOrder(ctx, db, id)
	if err != nil {
		return nil, err
	}
	if image != existing.Image {
		if err := deleteUnusedCoverImage(ctx, db, covers, existing.Image); err != nil {
			return nil, err
		}
	}
	return detail, nil
}

func rateReadingOrder(ctx context.Context, db *sqlx.DB, id int, rating float64) (*ReadingOrderDetailOutput, error) {
	if currentUserIsPublic(ctx) {
		return nil, huma.Error403Forbidden("read-only public access cannot rate reading orders")
	}
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if rating != 0 && (rating < 1 || rating > 5) {
		return nil, huma.Error400BadRequest("reading order rating must be 0 or between 1 and 5")
	}
	if _, err := getReadingOrderRow(ctx, db, id); err != nil {
		return nil, err
	}

	var exists int
	if err := db.GetContext(ctx, &exists, `SELECT COUNT(*) FROM reading_orders WHERE id = ?`, id); err != nil {
		return nil, huma.Error500InternalServerError("failed to check reading order")
	}
	if exists == 0 {
		return nil, huma.Error404NotFound("reading order not found")
	}

	if rating == 0 {
		if _, err := db.ExecContext(ctx, `
			DELETE FROM reading_order_ratings
			WHERE reading_order_id = ? AND user_id = ?
		`, id, userID); err != nil {
			return nil, huma.Error500InternalServerError("failed to clear reading order rating")
		}
		return getReadingOrder(ctx, db, id)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO reading_order_ratings (reading_order_id, user_id, rating, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(reading_order_id, user_id) DO UPDATE SET
			rating = excluded.rating,
			updated_at = excluded.updated_at
	`, id, userID, rating, currentTimestamp()); err != nil {
		return nil, huma.Error500InternalServerError("failed to rate reading order")
	}
	return getReadingOrder(ctx, db, id)
}

func copyReadingOrder(ctx context.Context, db *sqlx.DB, id int) (*ReadingOrderDetailOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if currentUserIsPublic(ctx) {
		return nil, huma.Error403Forbidden("read-only public access cannot copy reading orders")
	}

	source, err := getReadingOrderRow(ctx, db, id)
	if err != nil {
		return nil, err
	}
	entries, err := fetchReadingOrderEntries(ctx, db, id)
	if err != nil {
		return nil, err
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to start transaction")
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO reading_orders (name, description, image, favorite, is_public, author_user_id)
		VALUES (?, ?, ?, 0, ?, ?)
	`, copiedReadingOrderName(source.Name), source.Description, source.Image, source.IsPublic, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to copy reading order")
	}
	copiedID, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get copied reading order id")
	}

	for i, entry := range entries {
		position := i + 1
		switch entry.Type {
		case "comic":
			if entry.Comic == nil {
				continue
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO reading_order_comics (reading_order_id, comic_id, position, note, tags)
				VALUES (?, ?, ?, ?, ?)
			`, copiedID, entry.Comic.ID, position, entry.Comic.Comment, entry.Comic.Tags); err != nil {
				return nil, huma.Error500InternalServerError("failed to copy reading order comic")
			}
		case "readingOrder":
			if entry.ReadingOrder == nil {
				continue
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO reading_order_children (parent_reading_order_id, child_reading_order_id, position, note)
				VALUES (?, ?, ?, ?)
			`, copiedID, entry.ReadingOrder.ID, position, entry.Comment); err != nil {
				return nil, huma.Error500InternalServerError("failed to copy child reading order")
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, huma.Error500InternalServerError("failed to commit copied reading order")
	}
	return getReadingOrder(ctx, db, int(copiedID))
}

func copiedReadingOrderName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Copied reading order"
	}
	return name + " (Copy)"
}

func deleteReadingOrder(ctx context.Context, db *sqlx.DB, id int) (*struct{}, error) {
	if err := requireReadingOrderEditor(ctx, db, id); err != nil {
		return nil, err
	}
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
	return setReadingOrderComicsWithAuth(ctx, db, input, true)
}

func setReadingOrderComicsInternal(ctx context.Context, db *sqlx.DB, input *SetReadingOrderComicsInput) (*ReadingOrderDetailOutput, error) {
	return setReadingOrderComicsWithAuth(ctx, db, input, false)
}

func setReadingOrderComicsWithAuth(ctx context.Context, db *sqlx.DB, input *SetReadingOrderComicsInput, enforceAuthor bool) (*ReadingOrderDetailOutput, error) {
	ro, err := getReadingOrderRow(ctx, db, input.ID)
	if err != nil {
		return nil, err
	}
	if enforceAuthor {
		if err := requireReadingOrderEditor(ctx, db, input.ID); err != nil {
			return nil, err
		}
	}

	entries := readingOrderEntryItems(input)
	if err := validateReadingOrderEntries(entries); err != nil {
		return nil, err
	}
	if err := validateReadingOrderComicIDs(ctx, db, readingOrderEntryComicIDs(entries)); err != nil {
		return nil, err
	}
	if err := validateChildReadingOrderIDs(ctx, db, input.ID, readingOrderEntryChildIDs(entries)); err != nil {
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

	for i, entry := range entries {
		position := i + 1
		switch entry.Type {
		case "comic":
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO reading_order_comics (reading_order_id, comic_id, position, note, tags)
				VALUES (?, ?, ?, ?, ?)
			`, input.ID, entry.ComicID, position, entry.Comment, entry.Tags); err != nil {
				return nil, huma.Error500InternalServerError("failed to insert comic")
			}
		case "readingOrder":
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO reading_order_children (parent_reading_order_id, child_reading_order_id, position, note)
				VALUES (?, ?, ?, ?)
			`, input.ID, entry.ReadingOrderID, position, entry.Comment); err != nil {
				return nil, huma.Error500InternalServerError("failed to insert child reading order")
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, huma.Error500InternalServerError("failed to commit")
	}

	return fetchReadingOrderDetail(ctx, db, ro)
}

func requireReadingOrderEditor(ctx context.Context, db *sqlx.DB, id int) error {
	userID, err := currentUserID(ctx)
	if err != nil {
		return err
	}

	var row struct {
		AuthorID sql.NullInt64 `db:"author_user_id"`
		IsAdmin  bool          `db:"is_admin"`
	}
	if err := db.GetContext(ctx, &row, `
		SELECT ro.author_user_id, COALESCE(current_user.is_admin, 0) AS is_admin
		FROM reading_orders ro
		LEFT JOIN users current_user ON current_user.id = ?
		WHERE ro.id = ?
	`, userID, id); err != nil {
		if err == sql.ErrNoRows {
			return huma.Error404NotFound("reading order not found")
		}
		return huma.Error500InternalServerError("failed to check reading order author")
	}
	if row.IsAdmin {
		return nil
	}
	if !row.AuthorID.Valid || int(row.AuthorID.Int64) != userID {
		return huma.Error403Forbidden("only the reading order author or an admin can edit it")
	}
	return nil
}

func readingOrderEntryItems(input *SetReadingOrderComicsInput) []ReadingOrderEntryPayload {
	if len(input.Body.Entries) > 0 {
		entries := make([]ReadingOrderEntryPayload, 0, len(input.Body.Entries))
		for _, entry := range input.Body.Entries {
			entries = append(entries, normalizeReadingOrderEntry(entry))
		}
		return entries
	}

	comics := readingOrderComicItems(input)
	entries := make([]ReadingOrderEntryPayload, 0, len(comics)+len(input.Body.ReadingOrderIDs))
	for _, comic := range comics {
		entries = append(entries, ReadingOrderEntryPayload{
			Type:    "comic",
			ComicID: comic.ComicID,
			Comment: comic.Comment,
			Tags:    comic.Tags,
		})
	}
	for _, childID := range uniqueReadingOrderIDs(input.Body.ReadingOrderIDs) {
		entries = append(entries, ReadingOrderEntryPayload{
			Type:           "readingOrder",
			ReadingOrderID: childID,
			Comment:        "",
		})
	}
	return entries
}

func normalizeReadingOrderEntry(entry ReadingOrderEntryPayload) ReadingOrderEntryPayload {
	entry.Type = strings.TrimSpace(entry.Type)
	if entry.Type == "" {
		if entry.ReadingOrderID > 0 {
			entry.Type = "readingOrder"
		} else {
			entry.Type = "comic"
		}
	}
	return entry
}

func validateReadingOrderEntries(entries []ReadingOrderEntryPayload) error {
	for _, entry := range entries {
		switch entry.Type {
		case "comic":
			if entry.ComicID <= 0 {
				return huma.Error400BadRequest("comic entry requires comicId")
			}
		case "readingOrder":
			if entry.ReadingOrderID <= 0 {
				return huma.Error400BadRequest("reading order entry requires readingOrderId")
			}
		default:
			return huma.Error400BadRequest("reading order entry type must be comic or readingOrder")
		}
	}
	return nil
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

func readingOrderEntryComicIDs(entries []ReadingOrderEntryPayload) []int {
	comicIDs := make([]int, 0, len(entries))
	for _, entry := range entries {
		if entry.Type == "comic" {
			comicIDs = append(comicIDs, entry.ComicID)
		}
	}
	return comicIDs
}

func readingOrderEntryChildIDs(entries []ReadingOrderEntryPayload) []int {
	childIDs := make([]int, 0, len(entries))
	for _, entry := range entries {
		if entry.Type == "readingOrder" {
			childIDs = append(childIDs, entry.ReadingOrderID)
		}
	}
	return childIDs
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
	userID, err := currentUserID(ctx)
	if err != nil {
		return err
	}
	visibilityUserID := userID
	if currentUserIsPublic(ctx) {
		visibilityUserID = 0
	}

	query, args, err := sqlx.In(`
		SELECT id FROM reading_orders
		WHERE id IN (?) AND (
			is_public = 1
			OR author_user_id = ?
			OR EXISTS (SELECT 1 FROM users visibility_user WHERE visibility_user.id = ? AND visibility_user.is_admin = 1)
		)
	`, uniqueChildIDs, visibilityUserID, visibilityUserID)
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
