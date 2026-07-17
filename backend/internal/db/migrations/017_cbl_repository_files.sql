-- +goose Up
CREATE TABLE cbl_repository_files (
    repository_url         TEXT    NOT NULL,
    file_path              TEXT    NOT NULL,
    content_sha            TEXT    NOT NULL,
    reading_order_id       INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
    group_key              TEXT    NOT NULL DEFAULT '',
    imported_at            TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (repository_url, file_path)
);
CREATE INDEX idx_cbl_repository_files_reading_order
ON cbl_repository_files(reading_order_id);
-- +goose Down
DROP INDEX IF EXISTS idx_cbl_repository_files_reading_order;
DROP TABLE IF EXISTS cbl_repository_files;
