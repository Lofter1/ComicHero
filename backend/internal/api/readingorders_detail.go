package api

import (
	"context"
	"database/sql"
	"sort"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

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
	for i := range entries {
		entry := &entries[i]
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
		entry.Comics = childComics
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
		hydrateComicTitle(&comics[i].Comic)
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
	sections := []struct {
		ReadingOrderSection
		Position int `db:"position"`
	}{}
	if err := db.SelectContext(ctx, &sections, `
		SELECT title, description, position
		FROM reading_order_sections
		WHERE reading_order_id = ?
		ORDER BY position
	`, readingOrderID); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch reading order sections")
	}

	positioned := make([]struct {
		position int
		entry    ReadingOrderEntry
	}, 0, len(comics)+len(children)+len(sections))
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
	for i := range sections {
		section := sections[i].ReadingOrderSection
		positioned = append(positioned, struct {
			position int
			entry    ReadingOrderEntry
		}{
			position: sections[i].Position,
			entry: ReadingOrderEntry{
				Type:    "section",
				Section: &section,
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
