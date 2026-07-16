package api

import (
	"context"
	"errors"
	"strings"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func metronImportError(err error) error {
	if isContextCanceledError(err) {
		return err
	}
	return metronAPIError(err)
}

func isContextCanceledError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.Canceled) || strings.Contains(strings.ToLower(err.Error()), "context canceled")
}

func listMetronImportJobs(store *metronImportJobStore) *MetronImportJobListOutput {
	return &MetronImportJobListOutput{Body: store.list()}
}

func streamMetronImportJobs(ctx context.Context, store *metronImportJobStore, send func(MetronImportJobEvent) error) {
	jobs, unsubscribe := store.subscribe()
	defer unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			if err := send(MetronImportJobEvent{Job: job}); err != nil {
				return
			}
		}
	}
}

func getMetronImportJob(store *metronImportJobStore, id string) (*MetronImportJobOutput, error) {
	job, ok := store.get(id)
	if !ok {
		return nil, huma.Error404NotFound("import job not found")
	}
	return &MetronImportJobOutput{Body: job}, nil
}

func deleteMetronImportJob(store *metronImportJobStore, id string) (*struct{}, error) {
	if _, ok, deleted := store.deleteTerminal(id); !ok {
		return nil, huma.Error404NotFound("import job not found")
	} else if !deleted {
		return nil, huma.Error400BadRequest("only finished imports can be dismissed")
	}
	return nil, nil
}

func cancelMetronImportJob(store *metronImportJobStore, id string) (*MetronImportJobOutput, error) {
	job, ok := store.cancelJob(id)
	if !ok {
		return nil, huma.Error404NotFound("import job not found")
	}
	return &MetronImportJobOutput{Body: job}, nil
}

func continueMetronImportJob(ctx context.Context, store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, id string) (*MetronImportJobOutput, error) {
	job, ok := store.get(id)
	if !ok {
		return nil, huma.Error404NotFound("import job not found")
	}
	if job.Status != "canceled" {
		return nil, huma.Error400BadRequest("only canceled imports can be continued")
	}

	var next MetronImportJob
	switch job.Type {
	case "comic":
		next = startMetronComicImportWithOptions(ctx, store, db, client, covers, job.MetronID, job.Options)
	case "readingList":
		next = startMetronReadingListContinue(ctx, store, db, client, covers, job.MetronID, job.Options)
	case "readingLists":
		next = startAllMetronReadingListsImport(ctx, store, db, client, covers, job.Options)
	case "arc":
		next = startMetronArcContinue(ctx, store, db, client, covers, job.MetronID, job.Options)
	case "series":
		next = startMetronSeriesImportWithOptions(ctx, store, db, client, covers, job.MetronID, job.Options)
	case "character":
		next = startMetronCharacterAppearancesImportWithOptions(ctx, store, db, client, covers, job.MetronID, job.Options)
	default:
		return nil, huma.Error400BadRequest("unsupported import type")
	}
	return &MetronImportJobOutput{Body: next}, nil
}

func startMetronArcContinue(parent context.Context, store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int, options MetronImportOptions) MetronImportJob {
	options = resolveMetronImportOptions(options)
	return store.startWithContextAndOptions(parent, "arc", metronID, options, "Continuing arc import from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 0, "Fetching arc from Metron...")
		arc, info, err := fetchMetronArc(ctx, db, client, metronID, options.Force)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceArc, metronID); err != nil {
				return err
			}
			issues, err := client.GetArcIssues(ctx, metronID)
			if err != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				return metronImportError(err)
			}
			arc = &metron.MetronArc{ID: metronID, Issues: issues}
		}
		if err := importMetronArcWithOptions(ctx, db, client, covers, *arc, true, progress, options); err != nil {
			return err
		}
		return markMetronSynced(ctx, db, metronResourceArc, metronID, info)
	})
}

func startMetronReadingListContinue(parent context.Context, store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int, options MetronImportOptions) MetronImportJob {
	options = resolveMetronImportOptions(options)
	return store.startWithContextAndOptions(parent, "readingList", metronID, options, "Continuing reading list import from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		forceMetadata := options.Force
		if existingID, ok, err := existingReadingOrderIDByMetronID(ctx, db, metronID); err != nil {
			return err
		} else if ok && !forceMetadata {
			missingImage, err := readingOrderImageMissing(ctx, db, existingID)
			if err != nil {
				return err
			}
			forceMetadata = missingImage
		}

		progress(0, 0, "Fetching reading list from Metron...")
		list, info, err := fetchMetronReadingList(ctx, db, client, metronID, forceMetadata)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceReadingList, metronID); err != nil {
				return err
			}
			issues, err := client.GetReadingListIssues(ctx, metronID)
			if err != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				return metronImportError(err)
			}
			list = &metron.ReadingList{ID: metronID, Issues: issues}
		}
		if err := importMetronReadingListWithOptions(ctx, db, client, covers, *list, true, progress, options); err != nil {
			return err
		}
		return markMetronSynced(ctx, db, metronResourceReadingList, metronID, info)
	})
}
