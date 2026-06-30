-- +goose Up
ALTER TABLE arcs ADD COLUMN metron_arc_id INTEGER;
ALTER TABLE arcs ADD COLUMN image TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX idx_arcs_metron_arc_id
ON arcs(metron_arc_id)
WHERE metron_arc_id IS NOT NULL;

-- +goose Down
DROP INDEX idx_arcs_metron_arc_id;
ALTER TABLE arcs DROP COLUMN image;
ALTER TABLE arcs DROP COLUMN metron_arc_id;
