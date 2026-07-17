package api

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

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
			COALESCE(NULLIF(TRIM(ro.image), ''), (
				SELECT cover_comic.cover_image
				FROM reading_order_comics cover_entry
				JOIN comics cover_comic ON cover_comic.id = cover_entry.comic_id
				WHERE cover_entry.reading_order_id = ro.id
					AND TRIM(cover_comic.cover_image) <> ''
				ORDER BY cover_entry.position, cover_entry.rowid
				LIMIT 1
			), '') AS display_image,
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
