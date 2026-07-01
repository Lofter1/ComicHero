-- +goose Up
CREATE INDEX idx_comics_series_year_issue
ON comics(series, series_year, issue);

CREATE INDEX idx_comics_series_year_publisher
ON comics(series, series_year, publisher)
WHERE publisher <> '';

CREATE INDEX idx_comics_series_year_cover
ON comics(series, series_year, issue)
WHERE cover_image <> '';

-- +goose Down
DROP INDEX idx_comics_series_year_cover;
DROP INDEX idx_comics_series_year_publisher;
DROP INDEX idx_comics_series_year_issue;
