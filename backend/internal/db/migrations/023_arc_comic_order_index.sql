-- +goose Up
CREATE INDEX IF NOT EXISTS idx_arc_comics_arc_position
ON arc_comics(arc_id, position);

CREATE INDEX IF NOT EXISTS idx_arc_comics_comic_arc
ON arc_comics(comic_id, arc_id);

-- +goose Down
DROP INDEX IF EXISTS idx_arc_comics_comic_arc;
DROP INDEX IF EXISTS idx_arc_comics_arc_position;
