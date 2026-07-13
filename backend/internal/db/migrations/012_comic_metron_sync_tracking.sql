-- +goose Up
ALTER TABLE comics ADD COLUMN metron_synced_at TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_comics_metron_synced_at
ON comics(metron_synced_at)
WHERE metron_issue_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_comics_metron_synced_at;

CREATE TABLE comics_without_sync_tracking (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    series          TEXT    NOT NULL,
    series_year     INTEGER NOT NULL DEFAULT 0,
    issue           TEXT    NOT NULL,
    publisher       TEXT    NOT NULL,
    cover_date      TEXT    NOT NULL DEFAULT '',
    cover_image     TEXT    NOT NULL DEFAULT '',
    description     TEXT    NOT NULL DEFAULT '',
    read            INTEGER NOT NULL DEFAULT 0,
    metron_issue_id INTEGER,
    series_id       INTEGER REFERENCES series(id) ON DELETE SET NULL
);

INSERT INTO comics_without_sync_tracking (
    id, series, series_year, issue, publisher, cover_date, cover_image,
    description, read, metron_issue_id, series_id
)
SELECT
    id, series, series_year, issue, publisher, cover_date, cover_image,
    description, read, metron_issue_id, series_id
FROM comics;

DROP TABLE comics;
ALTER TABLE comics_without_sync_tracking RENAME TO comics;

CREATE UNIQUE INDEX idx_comics_metron_issue_id
ON comics(metron_issue_id)
WHERE metron_issue_id IS NOT NULL;

CREATE INDEX idx_comics_series_id_issue
ON comics(series_id, issue);
