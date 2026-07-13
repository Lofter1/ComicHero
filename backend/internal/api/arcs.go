package api

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func RegisterArcRoutes(api huma.API, db *sqlx.DB) {
	huma.Register(api, huma.Operation{
		OperationID: "listArcs",
		Tags:        []string{tagArcs},
		Summary:     "List arcs",
		Description: "Returns story arcs with computed read progress. Results can be filtered by text, favorite status, or a comic they contain.",
		Method:      http.MethodGet,
		Path:        "/arcs",
		Errors:      errsRead,
	}, func(ctx context.Context, input *ArcListInput) (*ArcListOutput, error) {
		return listArcs(ctx, db, input)
	})

	huma.Register(api, huma.Operation{
		OperationID: "getArc",
		Tags:        []string{tagArcs},
		Summary:     "Get an arc",
		Description: "Returns an arc by ID, including its comics in arc order and computed progress.",
		Method:      http.MethodGet,
		Path:        "/arcs/{id}",
		Errors:      errsRead,
	}, func(ctx context.Context, input *ArcInput) (*ArcDetailOutput, error) {
		return getArc(ctx, db, input.ID)
	})

	for _, operation := range []struct {
		id      string
		summary string
		method  string
		started bool
	}{{"startArc", "Start reading an arc", http.MethodPost, true}, {"stopArc", "Stop reading an arc", http.MethodDelete, false}} {
		op := operation
		huma.Register(api, huma.Operation{OperationID: op.id, Tags: []string{tagArcs}, Summary: op.summary, Method: op.method, Path: "/arcs/{id}/start", Errors: errsWrite}, func(ctx context.Context, input *ArcInput) (*ArcDetailOutput, error) {
			if err := setContentStarted(ctx, db, "user_arcs", "arc_id", "arcs", input.ID, op.started); err != nil {
				return nil, err
			}
			return getArc(ctx, db, input.ID)
		})
	}

	huma.Register(api, huma.Operation{
		OperationID:   "createArc",
		Tags:          []string{tagArcs},
		Summary:       "Create an arc",
		Description:   "Creates an arc with a name, description, and favorite flag.",
		Method:        http.MethodPost,
		Path:          "/arcs",
		DefaultStatus: 201,
		Errors:        errsWrite,
	}, func(ctx context.Context, input *CreateArcInput) (*CreateArcOutput, error) {
		return createArc(ctx, db, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateArc",
		Tags:        []string{tagArcs},
		Summary:     "Update an arc",
		Description: "Updates an arc's name, description, and favorite flag. It does not change the arc's comic entries.",
		Method:      http.MethodPut,
		Path:        "/arcs/{id}",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *UpdateArcInput) (*ArcDetailOutput, error) {
		return updateArc(ctx, db, input.ID, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "deleteArc",
		Tags:          []string{tagArcs},
		Summary:       "Delete an arc",
		Description:   "Deletes an arc by ID and clears its comic-entry and user-preference links. Admin access is required.",
		Method:        http.MethodDelete,
		Path:          "/arcs/{id}",
		DefaultStatus: 204,
		Errors:        errsWrite,
	}, func(ctx context.Context, input *ArcInput) (*struct{}, error) {
		return deleteArc(ctx, db, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "setArcComics",
		Tags:        []string{tagArcs},
		Summary:     "Set arc comics",
		Description: "Replaces every comic entry in an arc. Entry order is the submitted array order, duplicate comic IDs are allowed, and entries support per-entry comments.",
		Method:      http.MethodPut,
		Path:        "/arcs/{id}/comics",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *SetArcComicsInput) (*ArcDetailOutput, error) {
		return setArcComics(ctx, db, input)
	})
}

func listArcs(ctx context.Context, db *sqlx.DB, input *ArcListInput) (*ArcListOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	query, args, err := arcListQuery(input, userID)
	if err != nil {
		return nil, err
	}
	total, err := countRows(ctx, db, query, args)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to count arcs")
	}
	query, args, limit, offset := paginatedQuery(query, args, input.Limit, input.Offset)

	arcs := []Arc{}
	if err := db.SelectContext(ctx, &arcs, query, args...); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch arcs")
	}
	var pagination PaginationHeaders
	arcs, pagination = pageItems(arcs, limit, offset, total)
	return &ArcListOutput{PaginationHeaders: pagination, Body: arcs}, nil
}

func arcListQuery(input *ArcListInput, userID int) (string, []any, error) {
	query := newSelectQuery(`
		SELECT
			a.id,
			a.metron_arc_id,
			a.name,
			a.description,
			a.image,
			COALESCE(preference.favorite, 0) AS favorite,
			preference.started_at AS started_at,
			CASE
				WHEN COUNT(c.id) = 0 THEN 0.0
				ELSE CAST(SUM(CASE WHEN COALESCE(uc.read, 0) = 1 THEN 1 ELSE 0 END) AS REAL) / COUNT(c.id)
			END as progress
		FROM arcs a
		LEFT JOIN arc_comics ac ON ac.arc_id = a.id
		LEFT JOIN comics c ON c.id = ac.comic_id
		LEFT JOIN user_comics uc ON uc.comic_id = c.id AND uc.user_id = ?
		LEFT JOIN user_arcs preference ON preference.arc_id = a.id AND preference.user_id = ?
	`)
	query.args = append(query.args, userID, userID)

	if input.Query != "" {
		search := "%" + input.Query + "%"
		query.where("(a.name LIKE ? OR a.description LIKE ?)", search, search)
	}
	if favorite, ok, err := parseOptionalBool(input.Favorite, "favorite"); err != nil {
		return "", nil, err
	} else if ok {
		query.where("COALESCE(preference.favorite, 0) = ?", favorite)
	}
	if started, ok, err := parseOptionalBool(input.Started, "started"); err != nil {
		return "", nil, err
	} else if ok && started {
		query.where("preference.started_at IS NOT NULL")
	} else if ok {
		query.where("preference.started_at IS NULL")
	}
	if input.ComicID > 0 {
		query.where(`
			EXISTS (
				SELECT 1 FROM arc_comics matching_ac
				WHERE matching_ac.arc_id = a.id AND matching_ac.comic_id = ?
			)
		`, input.ComicID)
	}

	query.groupBy("GROUP BY a.id")
	query.orderBy(arcListOrder(input.Sort, input.Direction))
	sql, args := query.build()
	return sql, args, nil
}

func arcListOrder(sort, direction string) string {
	dir := sortDirection(direction)
	switch sort {
	case "progress":
		return "ORDER BY progress " + dir + ", a.name " + dir
	default:
		return "ORDER BY a.name " + dir
	}
}

func getArc(ctx context.Context, db *sqlx.DB, id int) (*ArcDetailOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	var arc Arc
	if err := db.GetContext(ctx, &arc, `
		SELECT a.id, a.metron_arc_id, a.name, a.description, a.image,
			COALESCE(preference.favorite, 0) AS favorite, preference.started_at AS started_at
		FROM arcs a
		LEFT JOIN user_arcs preference ON preference.arc_id = a.id AND preference.user_id = ?
		WHERE a.id = ?
	`, userID, id); err != nil {
		return nil, huma.Error404NotFound("arc not found")
	}

	return fetchArcDetail(ctx, db, arc)
}

func fetchArcDetail(ctx context.Context, db *sqlx.DB, arc Arc) (*ArcDetailOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	comics := []ArcComic{}
	if err := db.SelectContext(ctx, &comics, `
		SELECT c.*, COALESCE(uc.read, 0) AS read, COALESCE(uc.skipped, 0) AS skipped, ac.note AS comment FROM comics c
		JOIN arc_comics ac ON ac.comic_id = c.id
		LEFT JOIN user_comics uc ON uc.comic_id = c.id AND uc.user_id = ?
		WHERE ac.arc_id = ?
		ORDER BY ac.position
	`, userID, arc.ID); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch arc comics")
	}
	hydrateArcComicTitles(comics)

	arc.Progress = computeArcProgress(comics)
	return &ArcDetailOutput{
		Body: ArcDetail{
			Arc:    arc,
			Comics: comics,
		},
	}, nil
}

func createArc(ctx context.Context, db *sqlx.DB, payload ArcPayload) (*CreateArcOutput, error) {
	result, err := db.ExecContext(ctx, `
		INSERT INTO arcs (name, description)
		VALUES (?, ?)
	`, payload.Name, payload.Description)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create arc")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get new arc id")
	}
	if err := setContentFavorite(ctx, db, "user_arcs", "arc_id", "arcs", int(id), payload.Favorite); err != nil {
		return nil, err
	}
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	var arc Arc
	if err := db.GetContext(ctx, &arc, `
		SELECT a.id, a.metron_arc_id, a.name, a.description, a.image,
			COALESCE(preference.favorite, 0) AS favorite, preference.started_at AS started_at
		FROM arcs a
		LEFT JOIN user_arcs preference ON preference.arc_id = a.id AND preference.user_id = ?
		WHERE a.id = ?
	`, userID, id); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch created arc")
	}

	return &CreateArcOutput{Body: arc}, nil
}

func updateArc(ctx context.Context, db *sqlx.DB, id int, payload ArcPayload) (*ArcDetailOutput, error) {
	result, err := db.ExecContext(ctx, `
		UPDATE arcs
		SET name = ?, description = ?
		WHERE id = ?
	`, payload.Name, payload.Description, id)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to update arc")
	}
	if err := requireRowsAffected(result, "arc not found"); err != nil {
		return nil, err
	}
	if err := setContentFavorite(ctx, db, "user_arcs", "arc_id", "arcs", id, payload.Favorite); err != nil {
		return nil, err
	}

	return getArc(ctx, db, id)
}

func deleteArc(ctx context.Context, db *sqlx.DB, id int) (*struct{}, error) {
	if _, err := requireAdminUser(ctx, db); err != nil {
		return nil, err
	}
	result, err := db.ExecContext(ctx, `
		DELETE FROM arcs WHERE id = ?
	`, id)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to delete arc")
	}
	if err := requireRowsAffected(result, "arc not found"); err != nil {
		return nil, err
	}

	return &struct{}{}, nil
}

func setArcComics(ctx context.Context, db *sqlx.DB, input *SetArcComicsInput) (*ArcDetailOutput, error) {
	var arc Arc
	if err := db.GetContext(ctx, &arc, `
		SELECT * FROM arcs WHERE id = ?
	`, input.ID); err != nil {
		return nil, huma.Error404NotFound("arc not found")
	}

	comics := arcComicItems(input)
	if err := validateReadingOrderComicIDs(ctx, db, arcComicIDs(comics)); err != nil {
		return nil, err
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to start transaction")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM arc_comics WHERE arc_id = ?
	`, input.ID); err != nil {
		return nil, huma.Error500InternalServerError("failed to clear arc comics")
	}

	for i, comic := range comics {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO arc_comics (arc_id, comic_id, position, note)
			VALUES (?, ?, ?, ?)
		`, input.ID, comic.ComicID, i+1, comic.Comment); err != nil {
			return nil, huma.Error500InternalServerError("failed to insert arc comic")
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, huma.Error500InternalServerError("failed to commit arc")
	}

	return fetchArcDetail(ctx, db, arc)
}

func computeArcProgress(comics []ArcComic) float64 {
	if len(comics) == 0 {
		return 0
	}

	read := 0
	for _, comic := range comics {
		if comic.Read {
			read++
		}
	}
	return float64(read) / float64(len(comics))
}

func arcComicItems(input *SetArcComicsInput) []ArcComicPayload {
	if len(input.Body.Comics) > 0 {
		return input.Body.Comics
	}

	comics := make([]ArcComicPayload, 0, len(input.Body.ComicIDs))
	for _, comicID := range input.Body.ComicIDs {
		comics = append(comics, ArcComicPayload{ComicID: comicID})
	}
	return comics
}

func arcComicIDs(comics []ArcComicPayload) []int {
	comicIDs := make([]int, 0, len(comics))
	for _, comic := range comics {
		comicIDs = append(comicIDs, comic.ComicID)
	}
	return comicIDs
}

func hydrateArcComicTitles(comics []ArcComic) {
	for i := range comics {
		comics[i].Title = comicTitle(comics[i].Comic)
	}
}

func syncMetronIssueArcsWithOptions(ctx context.Context, db *sqlx.DB, client *metron.Client, comicID int, issue metron.Issue, options MetronImportOptions) error {
	options = resolveMetronImportOptions(options)
	if issue.Arcs == nil {
		return nil
	}
	if len(issue.Arcs) == 0 {
		if _, err := db.ExecContext(ctx, `DELETE FROM arc_comics WHERE comic_id = ?`, comicID); err != nil {
			return huma.Error500InternalServerError("failed to clear comic arcs")
		}
		return nil
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return huma.Error500InternalServerError("failed to start arc sync")
	}
	defer tx.Rollback()

	seen := map[int]bool{}
	arcIDs := make([]int, 0, len(issue.Arcs))
	for _, arc := range issue.Arcs {
		if options.includesArcs() && client != nil && arc.ID > 0 {
			detail, err := client.GetArcMetadata(ctx, arc.ID)
			if err != nil {
				if isContextCanceledError(err) {
					return err
				}
				return metronAPIError(err)
			}
			arc = *detail
		}
		id, err := upsertMetronIssueArc(ctx, tx, arc)
		if err != nil {
			return err
		}
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		arcIDs = append(arcIDs, id)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO arc_comics (arc_id, comic_id, position)
			SELECT ?, ?, 0
			WHERE NOT EXISTS (
				SELECT 1 FROM arc_comics WHERE arc_id = ? AND comic_id = ?
			)
		`, id, comicID, id, comicID); err != nil {
			return huma.Error500InternalServerError("failed to link comic arc")
		}
	}

	if len(arcIDs) > 0 {
		query, args, err := sqlx.In(`
			DELETE FROM arc_comics
			WHERE comic_id = ? AND arc_id NOT IN (?)
		`, comicID, arcIDs)
		if err != nil {
			return huma.Error500InternalServerError("failed to prepare stale arc cleanup")
		}
		query = tx.Rebind(query)
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return huma.Error500InternalServerError("failed to clear stale comic arcs")
		}
	}

	if err := tx.Commit(); err != nil {
		return huma.Error500InternalServerError("failed to save comic arcs")
	}
	return nil
}

func upsertMetronIssueArc(ctx context.Context, db sqlx.ExtContext, arc metron.MetronArc) (int, error) {
	if arc.ID > 0 {
		var id int
		if err := sqlx.GetContext(ctx, db, &id, `
			SELECT id FROM arcs WHERE metron_arc_id = ?
		`, arc.ID); err == nil {
			if arc.Name != "" {
				if _, err := db.ExecContext(ctx, `
					UPDATE arcs
					SET name = ?,
						description = CASE WHEN ? <> '' THEN ? ELSE description END,
						image = CASE WHEN ? <> '' THEN ? ELSE image END
					WHERE id = ?
				`, arc.Name, arc.Description, arc.Description, arc.Image, arc.Image, id); err != nil {
					return 0, huma.Error500InternalServerError("failed to update arc metadata")
				}
			}
			return id, nil
		} else if err != sql.ErrNoRows {
			return 0, huma.Error500InternalServerError("failed to check imported arc")
		}
	}

	if arc.Name == "" {
		return 0, nil
	}
	result, err := db.ExecContext(ctx, `
		INSERT INTO arcs (name, description, image, favorite, metron_arc_id)
		VALUES (?, ?, ?, ?, ?)
	`, arc.Name, arc.Description, arc.Image, false, nullableMetronID(arc.ID))
	if err != nil {
		return 0, huma.Error500InternalServerError("failed to save arc")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, huma.Error500InternalServerError("failed to get arc id")
	}
	return int(id), nil
}
