-- +goose Up
CREATE TABLE series (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL,
    series_year INTEGER NOT NULL DEFAULT 0,
    favorite    INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX idx_series_name_year
ON series(name, series_year);

INSERT OR IGNORE INTO series (name, series_year)
SELECT DISTINCT series, series_year
FROM comics
WHERE TRIM(series) <> '';

-- +goose Down
DROP TABLE series;
