package api

import (
	"context"
	"database/sql"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func importMetronArcWithOptions(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, arc metron.MetronArc, continueExisting bool, progress func(int, int, string), options MetronImportOptions) error {
	options = resolveMetronImportOptions(options)
	var arcID int
	if arc.ID > 0 {
		if id, ok, err := existingArcIDByMetronID(ctx, db, arc.ID); err != nil || ok {
			if ok {
				if !continueExisting {
					progress(1, 1, "Arc already exists.")
					return err
				}
				arcID = id
				if arc.Name != "" {
					if err := updateMetronArc(ctx, db, id, arc); err != nil {
						return err
					}
				}
			}
			if err != nil {
				return err
			}
		}
	}

	if arcID == 0 {
		created, err := createMetronArc(ctx, db, arc)
		if err != nil {
			return err
		}
		arcID = created.Body.ID
	}

	input := &SetArcComicsInput{ID: arcID}
	total := len(arc.Issues)
	progress(0, total, "Importing arc issues...")
	for i, issue := range arc.Issues {
		if err := ctx.Err(); err != nil {
			return err
		}
		comic, err := importMetronComicSweep(ctx, db, client, covers, issue, options, true)
		if err != nil {
			return err
		}
		input.Body.Comics = append(input.Body.Comics, ArcComicPayload{
			ComicID: comic.Body.ID,
		})
		progress(i+1, total, "Importing arc issues...")
	}

	if _, err := setArcComics(ctx, db, input); err != nil {
		return err
	}
	progress(total, total, "Arc imported.")
	return nil
}

func existingArcIDByMetronID(ctx context.Context, db *sqlx.DB, metronID int) (int, bool, error) {
	var id int
	if err := db.GetContext(ctx, &id, `
		SELECT id FROM arcs WHERE metron_arc_id = ?
	`, metronID); err != nil {
		if err != sql.ErrNoRows {
			return 0, false, huma.Error500InternalServerError("failed to check imported arc")
		}
		return 0, false, nil
	}
	return id, true, nil
}

func createMetronArc(ctx context.Context, db *sqlx.DB, arc metron.MetronArc) (*CreateArcOutput, error) {
	result, err := db.ExecContext(ctx, `
		INSERT INTO arcs (name, description, image, favorite, metron_arc_id)
		VALUES (?, ?, ?, ?, ?)
	`, arc.Name, arc.Description, arc.Image, false, nullableMetronID(arc.ID))
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to import Metron arc")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get imported arc id")
	}

	var local Arc
	if err := db.GetContext(ctx, &local, `
		SELECT * FROM arcs WHERE id = ?
	`, id); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch imported arc")
	}

	return &CreateArcOutput{Body: local}, nil
}

func updateMetronArc(ctx context.Context, db *sqlx.DB, id int, arc metron.MetronArc) error {
	result, err := db.ExecContext(ctx, `
		UPDATE arcs
		SET name = ?, description = ?, image = ?, metron_arc_id = ?
		WHERE id = ?
	`, arc.Name, arc.Description, arc.Image, nullableMetronID(arc.ID), id)
	if err != nil {
		return huma.Error500InternalServerError("failed to update Metron arc")
	}
	return requireRowsAffected(result, "arc not found")
}
