-- +goose Up
ALTER TABLE comics ADD COLUMN series_id INTEGER REFERENCES series(id) ON DELETE SET NULL;

INSERT OR IGNORE INTO series (name, series_year)
SELECT DISTINCT series, series_year
FROM comics
WHERE TRIM(series) <> '';

UPDATE comics
SET series_id = (
    SELECT id
    FROM series
    WHERE series.name = comics.series
      AND series.series_year = comics.series_year
    LIMIT 1
)
WHERE series_id IS NULL;

CREATE INDEX idx_comics_series_id_issue
ON comics(series_id, issue);

-- +goose Down
DROP INDEX IF EXISTS idx_comics_series_id_issue;

CREATE TABLE comics_without_series_id (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    series          TEXT    NOT NULL,
    series_year     INTEGER NOT NULL DEFAULT 0,
    issue           TEXT    NOT NULL,
    publisher       TEXT    NOT NULL,
    cover_date      TEXT    NOT NULL DEFAULT '',
    cover_image     TEXT    NOT NULL DEFAULT '',
    description     TEXT    NOT NULL DEFAULT '',
    metron_issue_id INTEGER
);

INSERT INTO comics_without_series_id (
    id,
    series,
    series_year,
    issue,
    publisher,
    cover_date,
    cover_image,
    description,
    metron_issue_id
)
SELECT
    id,
    series,
    series_year,
    issue,
    publisher,
    cover_date,
    cover_image,
    description,
    metron_issue_id
FROM comics;

DROP TABLE comics;

ALTER TABLE comics_without_series_id RENAME TO comics;

CREATE UNIQUE INDEX idx_comics_metron_issue_id
ON comics(metron_issue_id)
WHERE metron_issue_id IS NOT NULL;

CREATE INDEX idx_comics_series_year_issue
ON comics(series, series_year, issue);

CREATE INDEX idx_comics_series_year_publisher
ON comics(series, series_year, publisher)
WHERE publisher <> '';

CREATE INDEX idx_comics_series_year_cover
ON comics(series, series_year, issue)
WHERE cover_image <> '';
