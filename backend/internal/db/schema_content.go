package db

import "github.com/jmoiron/sqlx"

func ensureSeriesMetronSchema(db *sqlx.DB) error {
	exists, err := tableExists(db, "series")
	if err != nil || !exists {
		return err
	}

	hasMetronSeriesID, err := columnExists(db, "series", "metron_series_id")
	if err != nil {
		return err
	}
	if !hasMetronSeriesID {
		if _, err := db.Exec(`ALTER TABLE series ADD COLUMN metron_series_id INTEGER`); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_series_metron_series_id
		ON series(metron_series_id)
		WHERE metron_series_id IS NOT NULL
	`); err != nil {
		return err
	}
	return nil
}

func ensureComicSeriesLinks(db *sqlx.DB) error {
	comicsExists, err := tableExists(db, "comics")
	if err != nil || !comicsExists {
		return err
	}
	seriesExists, err := tableExists(db, "series")
	if err != nil || !seriesExists {
		return err
	}

	hasSeriesID, err := columnExists(db, "comics", "series_id")
	if err != nil {
		return err
	}
	if !hasSeriesID {
		if _, err := db.Exec(`ALTER TABLE comics ADD COLUMN series_id INTEGER REFERENCES series(id) ON DELETE SET NULL`); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`
		INSERT OR IGNORE INTO series (name, series_year)
		SELECT DISTINCT series, series_year
		FROM comics
		WHERE TRIM(series) <> ''
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		UPDATE comics
		SET series_id = (
			SELECT id
			FROM series
			WHERE series.name = comics.series
			  AND series.series_year = comics.series_year
			LIMIT 1
		)
		WHERE series_id IS NULL
		  AND TRIM(series) <> ''
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_comics_series_id_issue
		ON comics(series_id, issue)
	`); err != nil {
		return err
	}
	return nil
}
