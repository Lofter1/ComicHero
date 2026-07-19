-- +goose Up
CREATE INDEX IF NOT EXISTS idx_comic_characters_character_id
ON comic_characters(character_id, comic_id);

-- +goose Down
DROP INDEX IF EXISTS idx_comic_characters_character_id;
