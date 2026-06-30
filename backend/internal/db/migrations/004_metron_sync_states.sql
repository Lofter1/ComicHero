-- +goose Up
CREATE TABLE metron_sync_states (
    resource_type TEXT    NOT NULL,
    metron_id     INTEGER NOT NULL,
    last_modified TEXT    NOT NULL DEFAULT '',
    fully_synced  INTEGER NOT NULL DEFAULT 0,
    synced_at     TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (resource_type, metron_id)
);

-- +goose Down
DROP TABLE metron_sync_states;
