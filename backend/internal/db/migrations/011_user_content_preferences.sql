-- +goose Up
CREATE TABLE user_reading_orders_new (
    reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TEXT,
    favorite INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (reading_order_id, user_id)
);
INSERT INTO user_reading_orders_new (reading_order_id, user_id, started_at, favorite)
SELECT uro.reading_order_id, uro.user_id, uro.started_at, ro.favorite
FROM user_reading_orders uro JOIN reading_orders ro ON ro.id = uro.reading_order_id;
INSERT INTO user_reading_orders_new (reading_order_id, user_id, favorite)
SELECT ro.id, u.id, 1 FROM reading_orders ro CROSS JOIN users u WHERE ro.favorite = 1
ON CONFLICT(reading_order_id, user_id) DO UPDATE SET favorite = 1;
DROP TABLE user_reading_orders;
ALTER TABLE user_reading_orders_new RENAME TO user_reading_orders;

CREATE TABLE user_arcs (
    arc_id INTEGER NOT NULL REFERENCES arcs(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TEXT,
    favorite INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (arc_id, user_id)
);
INSERT INTO user_arcs (arc_id, user_id, started_at, favorite)
SELECT uas.arc_id, uas.user_id, uas.started_at, a.favorite
FROM user_arc_starts uas JOIN arcs a ON a.id = uas.arc_id;
INSERT INTO user_arcs (arc_id, user_id, favorite)
SELECT a.id, u.id, 1 FROM arcs a CROSS JOIN users u WHERE a.favorite = 1
ON CONFLICT(arc_id, user_id) DO UPDATE SET favorite = 1;
DROP TABLE user_arc_starts;

CREATE TABLE user_series (
    series_id INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TEXT,
    favorite INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (series_id, user_id)
);
INSERT INTO user_series (series_id, user_id, started_at, favorite)
SELECT uss.series_id, uss.user_id, uss.started_at, s.favorite
FROM user_series_starts uss JOIN series s ON s.id = uss.series_id;
INSERT INTO user_series (series_id, user_id, favorite)
SELECT s.id, u.id, 1 FROM series s CROSS JOIN users u WHERE s.favorite = 1
ON CONFLICT(series_id, user_id) DO UPDATE SET favorite = 1;
DROP TABLE user_series_starts;

CREATE TABLE user_characters (
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TEXT,
    favorite INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (character_id, user_id)
);
INSERT INTO user_characters (character_id, user_id, started_at, favorite)
SELECT ucs.character_id, ucs.user_id, ucs.started_at, c.favorite
FROM user_character_starts ucs JOIN characters c ON c.id = ucs.character_id;
INSERT INTO user_characters (character_id, user_id, favorite)
SELECT c.id, u.id, 1 FROM characters c CROSS JOIN users u WHERE c.favorite = 1
ON CONFLICT(character_id, user_id) DO UPDATE SET favorite = 1;
DROP TABLE user_character_starts;

CREATE INDEX idx_user_reading_orders_user_started ON user_reading_orders(user_id, started_at);
CREATE INDEX idx_user_reading_orders_user_favorite ON user_reading_orders(user_id, favorite);
CREATE INDEX idx_user_arcs_user_started ON user_arcs(user_id, started_at);
CREATE INDEX idx_user_arcs_user_favorite ON user_arcs(user_id, favorite);
CREATE INDEX idx_user_series_user_started ON user_series(user_id, started_at);
CREATE INDEX idx_user_series_user_favorite ON user_series(user_id, favorite);
CREATE INDEX idx_user_characters_user_started ON user_characters(user_id, started_at);
CREATE INDEX idx_user_characters_user_favorite ON user_characters(user_id, favorite);

-- +goose Down
CREATE TABLE user_reading_orders_old (
    reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (reading_order_id, user_id)
);
INSERT INTO user_reading_orders_old (reading_order_id, user_id, started_at)
SELECT reading_order_id, user_id, started_at
FROM user_reading_orders
WHERE started_at IS NOT NULL;
DROP TABLE user_reading_orders;
ALTER TABLE user_reading_orders_old RENAME TO user_reading_orders;

CREATE TABLE user_arc_starts (
    arc_id INTEGER NOT NULL REFERENCES arcs(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (arc_id, user_id)
);
INSERT INTO user_arc_starts (arc_id, user_id, started_at)
SELECT arc_id, user_id, started_at FROM user_arcs WHERE started_at IS NOT NULL;
DROP TABLE user_arcs;

CREATE TABLE user_series_starts (
    series_id INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (series_id, user_id)
);
INSERT INTO user_series_starts (series_id, user_id, started_at)
SELECT series_id, user_id, started_at FROM user_series WHERE started_at IS NOT NULL;
DROP TABLE user_series;

CREATE TABLE user_character_starts (
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (character_id, user_id)
);
INSERT INTO user_character_starts (character_id, user_id, started_at)
SELECT character_id, user_id, started_at FROM user_characters WHERE started_at IS NOT NULL;
DROP TABLE user_characters;

CREATE INDEX idx_user_reading_orders_user_started ON user_reading_orders(user_id, started_at);
CREATE INDEX idx_user_arc_starts_user_started ON user_arc_starts(user_id, started_at);
CREATE INDEX idx_user_series_starts_user_started ON user_series_starts(user_id, started_at);
CREATE INDEX idx_user_character_starts_user_started ON user_character_starts(user_id, started_at);
