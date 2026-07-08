package api

import (
	"context"
	"database/sql"
	"strings"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func importMetronReadingList(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, list metron.ReadingList) (*ReadingOrderDetailOutput, error) {
	if list.ID > 0 {
		if id, ok, err := existingReadingOrderIDByMetronID(ctx, db, list.ID); err != nil {
			return nil, err
		} else if ok {
			if err := updateMetronReadingOrderMetadata(ctx, db, covers, id, list); err != nil {
				return nil, err
			}
			return getReadingOrder(ctx, db, id)
		}
	}

	order, err := createMetronReadingOrder(ctx, db, covers, list)
	if err != nil {
		return nil, err
	}

	input := &SetReadingOrderComicsInput{ID: order.Body.ID}
	for _, issue := range list.Issues {
		comic, err := importMetronComic(ctx, db, client, covers, issue)
		if err != nil {
			return nil, err
		}
		input.Body.Comics = append(input.Body.Comics, ReadingOrderComicPayload{
			ComicID: comic.Body.ID,
			Tags:    strings.Join(issue.Tags, ", "),
		})
	}

	return setReadingOrderComicsInternal(ctx, db, input)
}

func continueMetronReadingListWithProgress(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, list metron.ReadingList, progress func(int, int, string)) error {
	return importMetronReadingListWithOptions(ctx, db, client, covers, list, true, progress, defaultMetronImportOptions())
}

func importMetronReadingListWithOptions(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, list metron.ReadingList, continueExisting bool, progress func(int, int, string), options MetronImportOptions) error {
	options = resolveMetronImportOptions(options)
	var orderID int
	if list.ID > 0 {
		if id, ok, err := existingReadingOrderIDByMetronID(ctx, db, list.ID); err != nil || ok {
			if ok {
				if !continueExisting {
					progress(1, 1, "Reading list already exists.")
					return err
				}
				orderID = id
			}
			if err != nil {
				return err
			}
		}
	}

	if orderID == 0 {
		order, err := createMetronReadingOrder(ctx, db, covers, list)
		if err != nil {
			return err
		}
		orderID = order.Body.ID
	} else if err := updateMetronReadingOrderMetadata(ctx, db, covers, orderID, list); err != nil {
		return err
	}

	input := &SetReadingOrderComicsInput{ID: orderID}
	total := len(list.Issues)
	progress(0, total, "Importing reading-list issues...")
	for i, issue := range list.Issues {
		if err := ctx.Err(); err != nil {
			return err
		}
		comic, err := importMetronComicSweep(ctx, db, client, covers, issue, options, true)
		if err != nil {
			return err
		}
		input.Body.Comics = append(input.Body.Comics, ReadingOrderComicPayload{
			ComicID: comic.Body.ID,
			Tags:    strings.Join(issue.Tags, ", "),
		})
		if _, err := setReadingOrderComicsInternal(ctx, db, input); err != nil {
			return err
		}
		progress(i+1, total, "Importing reading-list issues...")
	}

	if _, err := setReadingOrderComicsInternal(ctx, db, input); err != nil {
		return err
	}
	progress(total, total, "Reading list imported.")
	return nil
}

func existingReadingOrderIDByMetronID(ctx context.Context, db *sqlx.DB, metronID int) (int, bool, error) {
	var id int
	if err := db.GetContext(ctx, &id, `
		SELECT id FROM reading_orders WHERE metron_reading_list_id = ?
	`, metronID); err != nil {
		if err != sql.ErrNoRows {
			return 0, false, huma.Error500InternalServerError("failed to check imported reading list")
		}
		return 0, false, nil
	}
	return id, true, nil
}

func readingOrderImageMissing(ctx context.Context, db *sqlx.DB, id int) (bool, error) {
	var image string
	if err := db.GetContext(ctx, &image, `
		SELECT image FROM reading_orders WHERE id = ?
	`, id); err != nil {
		if err == sql.ErrNoRows {
			return false, huma.Error404NotFound("reading order not found")
		}
		return false, huma.Error500InternalServerError("failed to check reading order image")
	}
	return strings.TrimSpace(image) == "", nil
}

func createMetronReadingOrder(ctx context.Context, db *sqlx.DB, covers *CoverCache, list metron.ReadingList) (*CreateReadingOrderOutput, error) {
	image, err := localCoverURL(ctx, covers, list.Image)
	if err != nil {
		return nil, err
	}
	defaultUserID, err := ensureDefaultUser(ctx, db)
	if err != nil {
		return nil, err
	}

	result, err := db.ExecContext(ctx, `
		INSERT INTO reading_orders (name, description, image, favorite, metron_reading_list_id, author_user_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, list.Name, list.Description, image, false, nullableMetronID(list.ID), defaultUserID)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to import Metron reading list")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get imported reading order id")
	}

	ro, err := getReadingOrderRow(ctx, db, int(id))
	if err != nil {
		return nil, err
	}

	return &CreateReadingOrderOutput{Body: ro}, nil
}

func updateMetronReadingOrderMetadata(ctx context.Context, db *sqlx.DB, covers *CoverCache, id int, list metron.ReadingList) error {
	image, err := localCoverURL(ctx, covers, list.Image)
	if err != nil {
		return err
	}
	defaultUserID, err := ensureDefaultUser(ctx, db)
	if err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx, `
		UPDATE reading_orders
		SET name = COALESCE(NULLIF(?, ''), name),
			description = COALESCE(NULLIF(?, ''), description),
			image = COALESCE(NULLIF(?, ''), image),
			metron_reading_list_id = COALESCE(?, metron_reading_list_id),
			author_user_id = COALESCE(author_user_id, ?)
		WHERE id = ?
	`, list.Name, list.Description, image, nullableMetronID(list.ID), defaultUserID, id); err != nil {
		return huma.Error500InternalServerError("failed to update Metron reading list")
	}
	return nil
}
