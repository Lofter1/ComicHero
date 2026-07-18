-- +goose Up
CREATE TABLE character_collections (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT    NOT NULL COLLATE NOCASE,
    created_at TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TEXT
);

CREATE UNIQUE INDEX idx_character_collections_user_name
ON character_collections(user_id, name);

CREATE INDEX idx_character_collections_user_started
ON character_collections(user_id, started_at);

CREATE TABLE character_collection_members (
    collection_id INTEGER NOT NULL REFERENCES character_collections(id) ON DELETE CASCADE,
    character_id  INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    added_at      TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (collection_id, character_id)
);

CREATE INDEX idx_character_collection_members_character
ON character_collection_members(character_id);

-- +goose Down
DROP INDEX idx_character_collection_members_character;
DROP TABLE character_collection_members;
DROP INDEX idx_character_collections_user_started;
DROP INDEX idx_character_collections_user_name;
DROP TABLE character_collections;
