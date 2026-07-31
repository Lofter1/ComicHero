-- +goose Up
ALTER TABLE cbl_repository_files ADD COLUMN collection TEXT NOT NULL DEFAULT '';
ALTER TABLE cbl_repository_files ADD COLUMN collection_sequence INTEGER;

CREATE INDEX idx_cbl_repository_files_collection
ON cbl_repository_files(repository_url, collection);

-- +goose Down
DROP INDEX IF EXISTS idx_cbl_repository_files_collection;
ALTER TABLE cbl_repository_files DROP COLUMN collection_sequence;
ALTER TABLE cbl_repository_files DROP COLUMN collection;
