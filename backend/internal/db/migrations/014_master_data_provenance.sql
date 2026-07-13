-- +goose Up
ALTER TABLE comics ADD COLUMN created_at TEXT NOT NULL DEFAULT '';
ALTER TABLE comics ADD COLUMN created_by INTEGER REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE comics ADD COLUMN changed_at TEXT NOT NULL DEFAULT '';
ALTER TABLE comics ADD COLUMN changed_by INTEGER REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE reading_orders ADD COLUMN created_at TEXT NOT NULL DEFAULT '';
ALTER TABLE reading_orders ADD COLUMN created_by INTEGER REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE reading_orders ADD COLUMN changed_at TEXT NOT NULL DEFAULT '';
ALTER TABLE reading_orders ADD COLUMN changed_by INTEGER REFERENCES users(id) ON DELETE SET NULL;
UPDATE reading_orders SET created_by = author_user_id, changed_by = author_user_id;

ALTER TABLE arcs ADD COLUMN created_at TEXT NOT NULL DEFAULT '';
ALTER TABLE arcs ADD COLUMN created_by INTEGER REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE arcs ADD COLUMN changed_at TEXT NOT NULL DEFAULT '';
ALTER TABLE arcs ADD COLUMN changed_by INTEGER REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE series ADD COLUMN created_at TEXT NOT NULL DEFAULT '';
ALTER TABLE series ADD COLUMN created_by INTEGER REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE series ADD COLUMN changed_at TEXT NOT NULL DEFAULT '';
ALTER TABLE series ADD COLUMN changed_by INTEGER REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE characters ADD COLUMN created_at TEXT NOT NULL DEFAULT '';
ALTER TABLE characters ADD COLUMN created_by INTEGER REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE characters ADD COLUMN changed_at TEXT NOT NULL DEFAULT '';
ALTER TABLE characters ADD COLUMN changed_by INTEGER REFERENCES users(id) ON DELETE SET NULL;

UPDATE comics SET created_at = CURRENT_TIMESTAMP, changed_at = CURRENT_TIMESTAMP;
UPDATE reading_orders SET created_at = CURRENT_TIMESTAMP, changed_at = CURRENT_TIMESTAMP;
UPDATE arcs SET created_at = CURRENT_TIMESTAMP, changed_at = CURRENT_TIMESTAMP;
UPDATE series SET created_at = CURRENT_TIMESTAMP, changed_at = CURRENT_TIMESTAMP;
UPDATE characters SET created_at = CURRENT_TIMESTAMP, changed_at = CURRENT_TIMESTAMP;

CREATE TRIGGER comics_created_at AFTER INSERT ON comics BEGIN UPDATE comics SET created_at = CURRENT_TIMESTAMP, changed_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;
CREATE TRIGGER reading_orders_created_at AFTER INSERT ON reading_orders BEGIN UPDATE reading_orders SET created_at = CURRENT_TIMESTAMP, changed_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;
CREATE TRIGGER arcs_created_at AFTER INSERT ON arcs BEGIN UPDATE arcs SET created_at = CURRENT_TIMESTAMP, changed_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;
CREATE TRIGGER series_created_at AFTER INSERT ON series BEGIN UPDATE series SET created_at = CURRENT_TIMESTAMP, changed_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;
CREATE TRIGGER characters_created_at AFTER INSERT ON characters BEGIN UPDATE characters SET created_at = CURRENT_TIMESTAMP, changed_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;
CREATE TRIGGER comics_changed_at AFTER UPDATE ON comics WHEN NEW.changed_at = OLD.changed_at BEGIN UPDATE comics SET changed_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;
CREATE TRIGGER reading_orders_changed_at AFTER UPDATE ON reading_orders WHEN NEW.changed_at = OLD.changed_at BEGIN UPDATE reading_orders SET changed_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;
CREATE TRIGGER arcs_changed_at AFTER UPDATE ON arcs WHEN NEW.changed_at = OLD.changed_at BEGIN UPDATE arcs SET changed_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;
CREATE TRIGGER series_changed_at AFTER UPDATE ON series WHEN NEW.changed_at = OLD.changed_at BEGIN UPDATE series SET changed_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;
CREATE TRIGGER characters_changed_at AFTER UPDATE ON characters WHEN NEW.changed_at = OLD.changed_at BEGIN UPDATE characters SET changed_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;

-- +goose Down
DROP TRIGGER IF EXISTS characters_changed_at;
DROP TRIGGER IF EXISTS series_changed_at;
DROP TRIGGER IF EXISTS arcs_changed_at;
DROP TRIGGER IF EXISTS reading_orders_changed_at;
DROP TRIGGER IF EXISTS comics_changed_at;
DROP TRIGGER IF EXISTS characters_created_at;
DROP TRIGGER IF EXISTS series_created_at;
DROP TRIGGER IF EXISTS arcs_created_at;
DROP TRIGGER IF EXISTS reading_orders_created_at;
DROP TRIGGER IF EXISTS comics_created_at;
-- SQLite cannot drop columns safely while these tables participate in many
-- foreign-key relationships. The application only migrates forward in
-- production; restoring a pre-migration backup is the supported rollback.
