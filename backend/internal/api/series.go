package api

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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
		if err := authorizeMetron(ctx, db, metronScopeImport, "POST /series/{id}/metron/import"); err != nil {
			return nil, err
		}
		return importLocalSeriesFromMetron(ctx, db, client, covers, importJobs, input.ID)
	})
}

func listSeries(ctx context.Context, db *sqlx.DB, input *ComicSeriesListInput) (*ComicSeriesListOutput, error) {
	if err := syncSeriesRows(ctx, db); err != nil {
		return nil, err
	}

	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	query, args, err := seriesListQuery(input, userID)
	if err != nil {
		return nil, err
	}
	countQuery, countArgs, err := seriesListCountQuery(input)
	if err != nil {
		return nil, err
	}
	total, err := countRows(ctx, db, countQuery, countArgs)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to count series")
	}
	query, args, limit, offset := paginatedQuery(query, args, input.Limit, input.Offset)

	series := []ComicSeries{}
	if err := db.SelectContext(ctx, &series, query, args...); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch series")
	}
	var pagination PaginationHeaders
	series, pagination = pageItems(series, limit, offset, total)
	if err := hydrateSeriesPublishers(ctx, db, series); err != nil {
		return nil, err
	}
	return &ComicSeriesListOutput{PaginationHeaders: pagination, Body: series}, nil
}

func seriesListQuery(input *ComicSeriesListInput, userID int) (string, []any, error) {
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
			COALESCE(SUM(CASE WHEN COALESCE(uc.read, 0) = 1 THEN 1 ELSE 0 END), 0) AS read_count,
			CASE
				WHEN COUNT(c.id) = 0 THEN 0.0
				ELSE CAST(SUM(CASE WHEN COALESCE(uc.read, 0) = 1 THEN 1 ELSE 0 END) AS REAL) / COUNT(c.id)
			END AS progress,
			COALESCE((
				SELECT c2.cover_image
				FROM comics c2
				WHERE c2.series_id = s.id
					AND c2.cover_image <> ''
				ORDER BY CAST(c2.issue AS REAL), c2.issue
				LIMIT 1
			), '') AS cover_image
		FROM series s
		LEFT JOIN comics c ON c.series_id = s.id
		LEFT JOIN user_comics uc ON uc.comic_id = c.id AND uc.user_id = ?
	`)
	query.args = append(query.args, userID)

	if err := applySeriesListFilters(query, input); err != nil {
		return "", nil, err
	}

	query.groupBy("GROUP BY s.id")
	query.orderBy(seriesListOrder(input.Sort, input.Direction))
	sql, args := query.build()
	return sql, args, nil
}

func seriesListCountQuery(input *ComicSeriesListInput) (string, []any, error) {
	query := newSelectQuery(`SELECT s.id FROM series s`)
	if err := applySeriesListFilters(query, input); err != nil {
		return "", nil, err
	}
	sql, args := query.build()
	return sql, args, nil
}

func applySeriesListFilters(query *selectQuery, input *ComicSeriesListInput) error {
	if input.Query != "" {
		for _, token := range strings.Fields(input.Query) {
			search := "%" + token + "%"
			query.where(`(
				s.name LIKE ?
				OR CAST(s.series_year AS TEXT) LIKE ?
				OR s.publisher LIKE ?
				OR s.description LIKE ?
			)`, search, search, search, search)
		}
	}
	if favorite, ok, err := parseOptionalBool(input.Favorite, "favorite"); err != nil {
		return err
	} else if ok {
		query.where("s.favorite = ?", favorite)
	}
	return nil
}

func seriesListOrder(sort, direction string) string {
	dir := sortDirection(direction)
	switch sort {
	case "year":
		return "ORDER BY s.series_year " + dir + ", s.name " + dir
	case "publisher":
		return "ORDER BY s.publisher " + dir + ", s.name " + dir + ", s.series_year " + dir
	case "entries":
		return "ORDER BY entry_count " + dir + ", s.name " + dir + ", s.series_year " + dir
	case "progress":
		return "ORDER BY progress " + dir + ", s.name " + dir + ", s.series_year " + dir
	default:
		return "ORDER BY s.name " + dir + ", s.series_year " + dir
	}
}

func getSeries(ctx context.Context, db *sqlx.DB, id int) (*ComicSeriesDetailOutput, error) {
	series, err := getSeriesRow(ctx, db, id)
	if err != nil {
		return nil, err
	}

	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	comics := []Comic{}
	if err := db.SelectContext(ctx, &comics, `
		SELECT c.*, COALESCE(uc.read, 0) AS read FROM comics c
		LEFT JOIN user_comics uc ON uc.comic_id = c.id AND uc.user_id = ?
		WHERE c.series_id = ?
		ORDER BY c.series_year, CAST(c.issue AS REAL), c.issue
	`, userID, series.ID); err != nil {
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
	userID, err := currentUserID(ctx)
	if err != nil {
		return ComicSeries{}, err
	}
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
			COALESCE(SUM(CASE WHEN COALESCE(uc.read, 0) = 1 THEN 1 ELSE 0 END), 0) AS read_count,
			CASE
				WHEN COUNT(c.id) = 0 THEN 0.0
				ELSE CAST(SUM(CASE WHEN COALESCE(uc.read, 0) = 1 THEN 1 ELSE 0 END) AS REAL) / COUNT(c.id)
			END AS progress,
			COALESCE((
				SELECT c2.cover_image
				FROM comics c2
				WHERE c2.series_id = s.id
					AND c2.cover_image <> ''
				ORDER BY CAST(c2.issue AS REAL), c2.issue
				LIMIT 1
			), '') AS cover_image
		FROM series s
		LEFT JOIN comics c ON c.series_id = s.id
		LEFT JOIN user_comics uc ON uc.comic_id = c.id AND uc.user_id = ?
		WHERE s.id = ?
		GROUP BY s.id
	`, userID, id); err != nil {
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

func ensureSeriesRow(ctx context.Context, db sqlx.ExtContext, name string, year int) (int, error) {
	if name == "" {
		return 0, nil
	}
	if _, err := db.ExecContext(ctx, `
		INSERT OR IGNORE INTO series (name, series_year)
		VALUES (?, ?)
	`, name, year); err != nil {
		return 0, huma.Error500InternalServerError("failed to save series")
	}
	var id int
	if err := sqlx.GetContext(ctx, db, &id, `
		SELECT id FROM series
		WHERE name = ? AND series_year = ?
	`, name, year); err != nil {
		return 0, huma.Error500InternalServerError("failed to fetch series")
	}
	return id, nil
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

	job := startLocalSeriesMetronImport(ctx, importJobs, db, client, covers, id, metronID)
	return &MetronImportJobOutput{Body: job}, nil
}

func resolveMetronSeriesID(ctx context.Context, db *sqlx.DB, client *metron.Client, series ComicSeries) (int, error) {
	if series.MetronSeriesID != nil && *series.MetronSeriesID > 0 {
		return *series.MetronSeriesID, nil
	}

	matches, err := client.SearchSeries(ctx, metron.SeriesSearchOptions{Query: series.Name})
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
	if err := updateSeriesMetronMetadataFull(ctx, db, id, metadata); err != nil {
		if isConstraintError(err) {
			if fallbackErr := updateSeriesMetronMetadataPartial(ctx, db, id, metadata); fallbackErr != nil {
				return huma.Error500InternalServerError(fmt.Sprintf("failed to update series metadata: %v", fallbackErr))
			}
			return nil
		}
		return huma.Error500InternalServerError(fmt.Sprintf("failed to update series metadata: %v", err))
	}
	return nil
}

func updateSeriesMetronMetadataFull(ctx context.Context, db *sqlx.DB, id int, metadata metron.Series) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var previous struct {
		Name       string `db:"name"`
		SeriesYear int    `db:"series_year"`
	}
	if err := tx.GetContext(ctx, &previous, `SELECT name, series_year FROM series WHERE id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
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
	); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE comics
		SET series_id = ?,
			series = (SELECT name FROM series WHERE id = ?),
			series_year = (SELECT series_year FROM series WHERE id = ?)
		WHERE series_id = ?
			OR (series_id IS NULL AND series = ? AND series_year = ?)
	`, id, id, id, id, previous.Name, previous.SeriesYear)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func updateSeriesMetronMetadataPartial(ctx context.Context, db *sqlx.DB, id int, metadata metron.Series) error {
	_, err := db.ExecContext(ctx, `
		UPDATE series
		SET publisher = COALESCE(NULLIF(?, ''), publisher),
			volume = ?,
			year_end = ?,
			issue_count = ?,
			description = COALESCE(NULLIF(?, ''), description)
		WHERE id = ?
	`, metadata.Publisher,
		metadata.Volume,
		metadata.YearEnd,
		metadata.IssueCount,
		metadata.Description,
		id,
	)
	return err
}

func isConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "constraint")
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
	if err := updateSeriesMetronMetadata(ctx, db, id, metadata); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, `
		UPDATE comics
		SET series_id = ?
		WHERE series_id IS NULL
			AND series = ?
			AND series_year = ?
	`, id, metadata.Name, metadata.YearBegan)
	if err != nil {
		return huma.Error500InternalServerError("failed to link comics to series")
	}
	return nil
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
		UPDATE comics
		SET series_id = (
			SELECT id
			FROM series
			WHERE series.name = comics.series
				AND series.series_year = comics.series_year
			LIMIT 1
		)
		WHERE series_id IS NULL
			AND TRIM(series) <> ''
	`); err != nil {
		return huma.Error500InternalServerError("failed to link comics to series")
	}
	if _, err := db.ExecContext(ctx, `
		DELETE FROM series
		WHERE NOT EXISTS (
			SELECT 1 FROM comics c
			WHERE c.series_id = series.id
		)
			AND metron_series_id IS NULL
	`); err != nil {
		log.Printf("failed to prune empty series: %v", err)
	}
	return nil
}

func hydrateSeriesPublishers(ctx context.Context, db *sqlx.DB, series []ComicSeries) error {
	if len(series) == 0 {
		return nil
	}

	type publisherRow struct {
		SeriesID   int    `db:"series_id"`
		Publishers string `db:"publishers"`
	}
	ids := make([]int, 0, len(series))
	for i := range series {
		ids = append(ids, series[i].ID)
		if series[i].Publisher != "" {
			series[i].Publishers = []string{series[i].Publisher}
		}
	}

	query, args, err := sqlx.In(`
		SELECT series_id, GROUP_CONCAT(publisher, '||') AS publishers
		FROM (
			SELECT DISTINCT series_id, publisher
			FROM comics
			WHERE publisher <> '' AND series_id IN (?)
			ORDER BY publisher
		)
		GROUP BY series_id
	`, ids)
	if err != nil {
		return huma.Error500InternalServerError("failed to prepare series publishers")
	}
	query = db.Rebind(query)

	rows := []publisherRow{}
	if err := db.SelectContext(ctx, &rows, query, args...); err != nil {
		return huma.Error500InternalServerError("failed to fetch series publishers")
	}
	publishersBySeries := map[int][]string{}
	for _, row := range rows {
		publishersBySeries[row.SeriesID] = strings.Split(row.Publishers, "||")
	}
	for i := range series {
		if publishers := publishersBySeries[series[i].ID]; len(publishers) > 0 {
			series[i].Publishers = publishers
		}
	}
	return nil
}
