-- +goose Up
CREATE TABLE user_reading_orders (
    reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
    user_id          INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at       TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (reading_order_id, user_id)
);
CREATE INDEX idx_user_reading_orders_user_started
ON user_reading_orders(user_id, started_at);

CREATE TABLE user_arc_starts (
    arc_id     INTEGER NOT NULL REFERENCES arcs(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (arc_id, user_id)
);
CREATE INDEX idx_user_arc_starts_user_started ON user_arc_starts(user_id, started_at);

CREATE TABLE user_series_starts (
    series_id  INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (series_id, user_id)
);
CREATE INDEX idx_user_series_starts_user_started ON user_series_starts(user_id, started_at);

CREATE TABLE user_character_starts (
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (character_id, user_id)
);
CREATE INDEX idx_user_character_starts_user_started ON user_character_starts(user_id, started_at);

-- +goose Down
DROP INDEX IF EXISTS idx_user_character_starts_user_started;
DROP TABLE user_character_starts;
DROP INDEX IF EXISTS idx_user_series_starts_user_started;
DROP TABLE user_series_starts;
DROP INDEX IF EXISTS idx_user_arc_starts_user_started;
DROP TABLE user_arc_starts;
DROP INDEX IF EXISTS idx_user_reading_orders_user_started;
DROP TABLE user_reading_orders;
