package api

import (
	"context"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

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
		case "section":
			if entry.Section == nil {
				continue
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO reading_order_sections (reading_order_id, position, title, description)
				VALUES (?, ?, ?, ?)
			`, copiedID, position, entry.Section.Title, entry.Section.Description); err != nil {
				return nil, huma.Error500InternalServerError("failed to copy reading order section")
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
