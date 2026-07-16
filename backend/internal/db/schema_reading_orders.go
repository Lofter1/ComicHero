package db

import "github.com/jmoiron/sqlx"

func ensureReadingOrderAuthors(db *sqlx.DB) error {
	exists, err := tableExists(db, "reading_orders")
	if err != nil || !exists {
		return err
	}

	hasAuthor, err := columnExists(db, "reading_orders", "author_user_id")
	if err != nil {
		return err
	}
	if !hasAuthor {
		if _, err := db.Exec(`ALTER TABLE reading_orders ADD COLUMN author_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL`); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`
		UPDATE reading_orders
		SET author_user_id = (
			SELECT id
			FROM users
			WHERE is_default = 1 OR name = 'Default'
			ORDER BY is_default DESC, id
			LIMIT 1
		)
		WHERE author_user_id IS NULL
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_reading_orders_author_user_id
		ON reading_orders(author_user_id)
	`); err != nil {
		return err
	}
	return nil
}

func ensureReadingOrderRatings(db *sqlx.DB) error {
	exists, err := tableExists(db, "reading_orders")
	if err != nil || !exists {
		return err
	}

	hasRating, err := columnExists(db, "reading_orders", "rating")
	if err != nil {
		return err
	}
	if !hasRating {
		if _, err := db.Exec(`ALTER TABLE reading_orders ADD COLUMN rating REAL NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}

	hasRatingCount, err := columnExists(db, "reading_orders", "rating_count")
	if err != nil {
		return err
	}
	if !hasRatingCount {
		if _, err := db.Exec(`ALTER TABLE reading_orders ADD COLUMN rating_count INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_reading_orders_rating
		ON reading_orders(rating)
	`); err != nil {
		return err
	}
	return nil
}

func ensureReadingOrderUserRatings(db *sqlx.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS reading_order_ratings (
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			user_id          INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			rating           REAL    NOT NULL,
			created_at       TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at       TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (reading_order_id, user_id),
			CHECK (rating >= 1 AND rating <= 5)
		)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_reading_order_ratings_order
		ON reading_order_ratings(reading_order_id)
	`); err != nil {
		return err
	}
	return nil
}
