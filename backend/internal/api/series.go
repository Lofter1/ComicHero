package api

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func RegisterSeriesRoutes(api huma.API, db *sqlx.DB, client *metron.Client, covers *CoverCache, importJobs *metronImportJobStore) {
	huma.Register(api, huma.Operation{
		OperationID: "listSeries",
		Tags:        []string{tagSeries},
		Summary:     "List series",
		Description: "Returns local series with favorite state, computed read progress, entry counts, publishers, and a representative cover image.",
		Method:      http.MethodGet,
		Path:        "/series",
		Errors:      errsRead,
	}, func(ctx context.Context, input *ComicSeriesListInput) (*ComicSeriesListOutput, error) {
		return listSeries(ctx, db, input)
	})

	huma.Register(api, huma.Operation{
		OperationID: "getSeries",
		Tags:        []string{tagSeries},
		Summary:     "Get a series",
		Description: "Returns a series by ID, including local comic entries ordered by series year and issue number.",
		Method:      http.MethodGet,
		Path:        "/series/{id}",
		Errors:      errsRead,
	}, func(ctx context.Context, input *ComicSeriesInput) (*ComicSeriesDetailOutput, error) {
		return getSeries(ctx, db, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateSeriesFavorite",
		Tags:        []string{tagSeries},
		Summary:     "Update series favorite status",
		Description: "Marks or unmarks a series as a favorite without changing its comic entries.",
		Method:      http.MethodPatch,
		Path:        "/series/{id}/favorite",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *UpdateComicSeriesFavoriteInput) (*ComicSeriesDetailOutput, error) {
		return updateSeriesFavorite(ctx, db, input.ID, input.Body.Favorite)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "importSeriesFromMetron",
		Tags:          []string{tagSeries, tagMetron},
		Summary:       "Import series from Metron",
		Description:   "Fetches Metron series metadata, saves it to the local series, then imports or reuses missing comics from the Metron series issue list.",
		Method:        http.MethodPost,
		Path:          "/series/{id}/metron/import",
		DefaultStatus: http.StatusAccepted,
		Errors:        errsMetronSync,
	}, func(ctx context.Context, input *ComicSeriesInput) (*MetronImportJobOutput, error) {
		return importLocalSeriesFromMetron(ctx, db, client, covers, importJobs, input.ID)
	})
}

func listSeries(ctx context.Context, db *sqlx.DB, input *ComicSeriesListInput) (*ComicSeriesListOutput, error) {
	if err := syncSeriesRows(ctx, db); err != nil {
		return nil, err
	}

	query, args, err := seriesListQuery(input)
	if err != nil {
		return nil, err
	}

	series := []ComicSeries{}
	if err := db.SelectContext(ctx, &series, query, args...); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch series")
	}
	if err := hydrateSeriesPublishers(ctx, db, series); err != nil {
		return nil, err
	}
	return &ComicSeriesListOutput{Body: series}, nil
}

func seriesListQuery(input *ComicSeriesListInput) (string, []any, error) {
	query := newSelectQuery(`
		SELECT
			s.id,
			s.metron_series_id,
			s.name,
			s.series_year,
			s.favorite,
			s.publisher,
			s.volume,
			s.year_end,
			s.issue_count,
			s.description,
			COUNT(c.id) AS entry_count,
			COALESCE(SUM(CASE WHEN c.read = 1 THEN 1 ELSE 0 END), 0) AS read_count,
			CASE
				WHEN COUNT(c.id) = 0 THEN 0.0
				ELSE CAST(SUM(CASE WHEN c.read = 1 THEN 1 ELSE 0 END) AS REAL) / COUNT(c.id)
			END AS progress,
			COALESCE((
				SELECT c2.cover_image
				FROM comics c2
				WHERE c2.series = s.name
					AND c2.series_year = s.series_year
					AND c2.cover_image <> ''
				ORDER BY c2.issue
				LIMIT 1
			), '') AS cover_image
		FROM series s
		LEFT JOIN comics c ON c.series = s.name AND c.series_year = s.series_year
	`)

	if input.Query != "" {
		search := "%" + input.Query + "%"
		query.where(`(
			s.name LIKE ?
			OR CAST(s.series_year AS TEXT) LIKE ?
			OR EXISTS (
				SELECT 1 FROM comics matching_c
				WHERE matching_c.series = s.name
					AND matching_c.series_year = s.series_year
					AND (
						matching_c.publisher LIKE ?
						OR CAST(matching_c.issue AS TEXT) LIKE ?
						OR matching_c.cover_date LIKE ?
					)
			)
		)`, search, search, search, search, search)
	}
	if favorite, ok, err := parseOptionalBool(input.Favorite, "favorite"); err != nil {
		return "", nil, err
	} else if ok {
		query.where("s.favorite = ?", favorite)
	}

	query.groupBy("GROUP BY s.id")
	query.orderBy("ORDER BY s.name, s.series_year")
	sql, args := query.build()
	return sql, args, nil
}

func getSeries(ctx context.Context, db *sqlx.DB, id int) (*ComicSeriesDetailOutput, error) {
	if err := syncSeriesRows(ctx, db); err != nil {
		return nil, err
	}

	series, err := getSeriesRow(ctx, db, id)
	if err != nil {
		return nil, err
	}

	comics := []Comic{}
	if err := db.SelectContext(ctx, &comics, `
		SELECT c.* FROM comics c
		WHERE c.series = ? AND c.series_year = ?
		ORDER BY c.series_year, c.issue
	`, series.Name, series.SeriesYear); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch series entries")
	}
	hydrateComicTitles(comics)

	return &ComicSeriesDetailOutput{
		Body: ComicSeriesDetail{
			ComicSeries: series,
			Comics:      comics,
		},
	}, nil
}

func getSeriesRow(ctx context.Context, db *sqlx.DB, id int) (ComicSeries, error) {
	var series ComicSeries
	if err := db.GetContext(ctx, &series, `
		SELECT
			s.id,
			s.metron_series_id,
			s.name,
			s.series_year,
			s.favorite,
			s.publisher,
			s.volume,
			s.year_end,
			s.issue_count,
			s.description,
			COUNT(c.id) AS entry_count,
			COALESCE(SUM(CASE WHEN c.read = 1 THEN 1 ELSE 0 END), 0) AS read_count,
			CASE
				WHEN COUNT(c.id) = 0 THEN 0.0
				ELSE CAST(SUM(CASE WHEN c.read = 1 THEN 1 ELSE 0 END) AS REAL) / COUNT(c.id)
			END AS progress,
			COALESCE((
				SELECT c2.cover_image
				FROM comics c2
				WHERE c2.series = s.name
					AND c2.series_year = s.series_year
					AND c2.cover_image <> ''
				ORDER BY c2.issue
				LIMIT 1
			), '') AS cover_image
		FROM series s
		LEFT JOIN comics c ON c.series = s.name AND c.series_year = s.series_year
		WHERE s.id = ?
		GROUP BY s.id
	`, id); err != nil {
		if err == sql.ErrNoRows {
			return ComicSeries{}, huma.Error404NotFound("series not found")
		}
		return ComicSeries{}, huma.Error500InternalServerError("failed to fetch series")
	}
	seriesList := []ComicSeries{series}
	if err := hydrateSeriesPublishers(ctx, db, seriesList); err != nil {
		return ComicSeries{}, err
	}
	return seriesList[0], nil
}

func updateSeriesFavorite(ctx context.Context, db *sqlx.DB, id int, favorite bool) (*ComicSeriesDetailOutput, error) {
	result, err := db.ExecContext(ctx, `
		UPDATE series
		SET favorite = ?
		WHERE id = ?
	`, favorite, id)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to update series favorite")
	}
	if err := requireRowsAffected(result, "series not found"); err != nil {
		return nil, err
	}
	return getSeries(ctx, db, id)
}

func ensureSeriesRow(ctx context.Context, db sqlx.ExtContext, name string, year int) error {
	if name == "" {
		return nil
	}
	if _, err := db.ExecContext(ctx, `
		INSERT OR IGNORE INTO series (name, series_year)
		VALUES (?, ?)
	`, name, year); err != nil {
		return huma.Error500InternalServerError("failed to save series")
	}
	return nil
}

func importLocalSeriesFromMetron(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, importJobs *metronImportJobStore, id int) (*MetronImportJobOutput, error) {
	series, err := getSeriesRow(ctx, db, id)
	if err != nil {
		return nil, err
	}

	metronID, err := resolveMetronSeriesID(ctx, db, client, series)
	if err != nil {
		return nil, err
	}

	job := startLocalSeriesMetronImport(importJobs, db, client, covers, id, metronID)
	return &MetronImportJobOutput{Body: job}, nil
}

func resolveMetronSeriesID(ctx context.Context, db *sqlx.DB, client *metron.Client, series ComicSeries) (int, error) {
	if series.MetronSeriesID != nil && *series.MetronSeriesID > 0 {
		return *series.MetronSeriesID, nil
	}

	matches, err := client.SearchSeries(ctx, series.Name)
	if err != nil {
		return 0, metronAPIError(err)
	}
	for _, match := range matches {
		if metronSeriesMatchesLocal(match, series) {
			if err := updateSeriesMetronMetadata(ctx, db, series.ID, match); err != nil {
				return 0, err
			}
			return match.ID, nil
		}
	}
	if len(matches) > 0 && matches[0].ID > 0 {
		if err := updateSeriesMetronMetadata(ctx, db, series.ID, matches[0]); err != nil {
			return 0, err
		}
		return matches[0].ID, nil
	}
	return 0, huma.Error404NotFound("matching Metron series not found")
}

func metronSeriesMatchesLocal(candidate metron.Series, series ComicSeries) bool {
	if candidate.ID <= 0 {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(candidate.Name), strings.TrimSpace(series.Name)) {
		return false
	}
	return series.SeriesYear == 0 || candidate.YearBegan == 0 || candidate.YearBegan == series.SeriesYear
}

func updateSeriesMetronMetadata(ctx context.Context, db *sqlx.DB, id int, metadata metron.Series) error {
	if metadata.ID <= 0 {
		return nil
	}
	_, err := db.ExecContext(ctx, `
		UPDATE series
		SET metron_series_id = ?,
			name = COALESCE(NULLIF(?, ''), name),
			series_year = CASE WHEN ? > 0 THEN ? ELSE series_year END,
			publisher = COALESCE(NULLIF(?, ''), publisher),
			volume = ?,
			year_end = ?,
			issue_count = ?,
			description = COALESCE(NULLIF(?, ''), description)
		WHERE id = ?
	`, metadata.ID,
		metadata.Name,
		metadata.YearBegan,
		metadata.YearBegan,
		metadata.Publisher,
		metadata.Volume,
		metadata.YearEnd,
		metadata.IssueCount,
		metadata.Description,
		id,
	)
	if err != nil {
		return huma.Error500InternalServerError("failed to update series metadata")
	}
	return nil
}

func updateImportedSeriesMetadata(ctx context.Context, db *sqlx.DB, metadata metron.Series) error {
	if metadata.ID <= 0 {
		return nil
	}
	if _, err := db.ExecContext(ctx, `
		INSERT OR IGNORE INTO series (name, series_year, metron_series_id, publisher, volume, year_end, issue_count, description)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, metadata.Name,
		metadata.YearBegan,
		metadata.ID,
		metadata.Publisher,
		metadata.Volume,
		metadata.YearEnd,
		metadata.IssueCount,
		metadata.Description,
	); err != nil {
		return huma.Error500InternalServerError("failed to save series metadata")
	}

	var id int
	if err := db.GetContext(ctx, &id, `
		SELECT id FROM series
		WHERE metron_series_id = ?
			OR (name = ? AND series_year = ?)
		ORDER BY CASE WHEN metron_series_id = ? THEN 0 ELSE 1 END, id
		LIMIT 1
	`, metadata.ID, metadata.Name, metadata.YearBegan, metadata.ID); err != nil {
		return huma.Error500InternalServerError("failed to fetch imported series")
	}
	return updateSeriesMetronMetadata(ctx, db, id, metadata)
}

func syncSeriesRows(ctx context.Context, db *sqlx.DB) error {
	if _, err := db.ExecContext(ctx, `
		INSERT OR IGNORE INTO series (name, series_year)
		SELECT DISTINCT series, series_year
		FROM comics
		WHERE TRIM(series) <> ''
	`); err != nil {
		return huma.Error500InternalServerError("failed to sync series")
	}
	if _, err := db.ExecContext(ctx, `
		DELETE FROM series
		WHERE NOT EXISTS (
			SELECT 1 FROM comics c
			WHERE c.series = series.name AND c.series_year = series.series_year
		)
	`); err != nil {
		return huma.Error500InternalServerError("failed to prune empty series")
	}
	return nil
}

func hydrateSeriesPublishers(ctx context.Context, db *sqlx.DB, series []ComicSeries) error {
	if len(series) == 0 {
		return nil
	}

	for i := range series {
		var publishers []string
		if err := db.SelectContext(ctx, &publishers, `
			SELECT DISTINCT publisher
			FROM comics
			WHERE series = ? AND series_year = ? AND publisher <> ''
			ORDER BY publisher
		`, series[i].Name, series[i].SeriesYear); err != nil {
			return huma.Error500InternalServerError("failed to fetch series publishers")
		}
		series[i].Publishers = publishers
	}
	return nil
}
