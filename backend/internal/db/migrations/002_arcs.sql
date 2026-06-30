-- +goose Up
CREATE TABLE arcs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    favorite    INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE arc_comics (
    arc_id   INTEGER NOT NULL REFERENCES arcs(id) ON DELETE CASCADE,
    comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
    position INTEGER NOT NULL DEFAULT 0,
    note     TEXT    NOT NULL DEFAULT ''
);

-- +goose Down
DROP TABLE arc_comics;
DROP TABLE arcs;
