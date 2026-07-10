package api

import (
	"context"
	"fmt"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func setContentStarted(ctx context.Context, db *sqlx.DB, table, idColumn, entityTable string, id int, started bool) error {
	userID, err := currentUserID(ctx)
	if err != nil {
		return err
	}
	var exists int
	if err := db.GetContext(ctx, &exists, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE id = ?", entityTable), id); err != nil {
		return huma.Error500InternalServerError("failed to check item")
	}
	if exists == 0 {
		return huma.Error404NotFound("item not found")
	}
	if started {
		_, err = db.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO %s (%s, user_id, started_at)
			VALUES (?, ?, ?)
			ON CONFLICT(%s, user_id) DO UPDATE SET started_at = excluded.started_at
		`, table, idColumn, idColumn), id, userID, time.Now().UTC().Format(time.RFC3339))
	} else {
		_, err = db.ExecContext(ctx, fmt.Sprintf(`
			UPDATE %s SET started_at = NULL WHERE %s = ? AND user_id = ?;
			DELETE FROM %s WHERE %s = ? AND user_id = ? AND favorite = 0
		`, table, idColumn, table, idColumn), id, userID, id, userID)
	}
	if err != nil {
		return huma.Error500InternalServerError("failed to update started state")
	}
	return nil
}

func setContentFavorite(ctx context.Context, db *sqlx.DB, table, idColumn, entityTable string, id int, favorite bool) error {
	userID, err := currentUserID(ctx)
	if err != nil {
		return err
	}
	var exists int
	if err := db.GetContext(ctx, &exists, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE id = ?", entityTable), id); err != nil {
		return huma.Error500InternalServerError("failed to check item")
	}
	if exists == 0 {
		return huma.Error404NotFound("item not found")
	}
	if favorite {
		_, err = db.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO %s (%s, user_id, favorite) VALUES (?, ?, 1)
			ON CONFLICT(%s, user_id) DO UPDATE SET favorite = 1
		`, table, idColumn, idColumn), id, userID)
	} else {
		_, err = db.ExecContext(ctx, fmt.Sprintf(`
			UPDATE %s SET favorite = 0 WHERE %s = ? AND user_id = ?;
			DELETE FROM %s WHERE %s = ? AND user_id = ? AND started_at IS NULL
		`, table, idColumn, table, idColumn), id, userID, id, userID)
	}
	if err != nil {
		return huma.Error500InternalServerError("failed to update favorite state")
	}
	return nil
}
