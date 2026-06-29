-- +goose Up
CREATE TABLE reading_orders (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    name                   TEXT    NOT NULL,
    description            TEXT    NOT NULL DEFAULT '',
    favorite               INTEGER NOT NULL DEFAULT 0,
    metron_reading_list_id INTEGER
);
CREATE UNIQUE INDEX idx_reading_orders_metron_reading_list_id
ON reading_orders(metron_reading_list_id)
WHERE metron_reading_list_id IS NOT NULL;

CREATE TABLE comics (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    series          TEXT    NOT NULL,
    series_year     INTEGER NOT NULL DEFAULT 0,
    issue           INTEGER NOT NULL,
    publisher       TEXT    NOT NULL,
    cover_date      TEXT    NOT NULL DEFAULT '',
    cover_image     TEXT    NOT NULL DEFAULT '',
    description     TEXT    NOT NULL DEFAULT '',
    read            INTEGER NOT NULL DEFAULT 0,
    metron_issue_id INTEGER
);
CREATE UNIQUE INDEX idx_comics_metron_issue_id
ON comics(metron_issue_id)
WHERE metron_issue_id IS NOT NULL;

CREATE TABLE series (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    name             TEXT    NOT NULL,
    series_year      INTEGER NOT NULL DEFAULT 0,
    favorite         INTEGER NOT NULL DEFAULT 0,
    metron_series_id INTEGER,
    publisher        TEXT    NOT NULL DEFAULT '',
    volume           INTEGER NOT NULL DEFAULT 0,
    year_end         INTEGER NOT NULL DEFAULT 0,
    issue_count      INTEGER NOT NULL DEFAULT 0,
    description      TEXT    NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX idx_series_name_year
ON series(name, series_year);
CREATE UNIQUE INDEX idx_series_metron_series_id
ON series(metron_series_id)
WHERE metron_series_id IS NOT NULL;

INSERT OR IGNORE INTO series (name, series_year)
SELECT DISTINCT series, series_year
FROM comics
WHERE TRIM(series) <> '';

CREATE TABLE characters (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    name                TEXT    NOT NULL,
    description         TEXT    NOT NULL DEFAULT '',
    image               TEXT    NOT NULL DEFAULT '',
    favorite            INTEGER NOT NULL DEFAULT 0,
    metron_character_id INTEGER
);
CREATE UNIQUE INDEX idx_characters_metron_character_id
ON characters(metron_character_id)
WHERE metron_character_id IS NOT NULL;

CREATE TABLE character_aliases (
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    alias        TEXT    NOT NULL,
    PRIMARY KEY (character_id, alias)
);

CREATE TABLE comic_characters (
    comic_id     INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    PRIMARY KEY (comic_id, character_id)
);

CREATE TABLE reading_order_comics (
    reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
    comic_id         INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
    position         INTEGER NOT NULL DEFAULT 0,
    note             TEXT    NOT NULL DEFAULT ''
);

-- +goose Down
DROP TABLE reading_order_comics;
DROP TABLE comic_characters;
DROP TABLE character_aliases;
DROP TABLE characters;
DROP TABLE series;
DROP TABLE comics;
DROP TABLE reading_orders;