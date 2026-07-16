package api

import (
	"context"
	"fmt"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/jmoiron/sqlx"
)

func startMetronComicImportWithOptions(parent context.Context, store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int, options MetronImportOptions) MetronImportJob {
	options = resolveMetronImportOptions(options)
	return store.startWithContextAndOptions(parent, "comic", metronID, options, "Importing comic from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 1, "Checking existing imports...")
		if _, ok, err := existingComicIDByMetronIssueID(ctx, db, metronID); err != nil || ok {
			if ok && options.Mode != "full" && !options.Force {
				progress(1, 1, "Comic already exists.")
				return err
			}
			if err != nil {
				return err
			}
		}

		progress(0, 1, "Fetching comic from Metron...")
		issue, info, err := fetchMetronIssue(ctx, db, client, metronID, options.Force)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceIssue, metronID); err != nil {
				return err
			}
			if _, ok, err := existingComicIDByMetronIssueID(ctx, db, metronID); err != nil || ok {
				progress(1, 1, "Comic metadata already current.")
				return err
			}
			issue, info, err = fetchMetronIssue(ctx, db, client, metronID, true)
			if err != nil {
				return metronImportError(err)
			}
		}
		_, err = importMetronComicSweep(ctx, db, client, covers, *issue, options, false)
		if err == nil {
			if err := markMetronSynced(ctx, db, metronResourceIssue, metronID, info); err != nil {
				return err
			}
			progress(1, 1, "Comic imported.")
		}
		return err
	})
}

func startMetronReadingListImportWithOptions(parent context.Context, store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int, options MetronImportOptions) MetronImportJob {
	options = resolveMetronImportOptions(options)
	return store.startWithContextAndOptions(parent, "readingList", metronID, options, "Importing reading list and issues from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 0, "Checking existing imports...")
		existingID, existing, err := existingReadingOrderIDByMetronID(ctx, db, metronID)
		if err != nil || existing {
			if existing && options.Mode != "full" && !options.Force {
				progress(1, 1, "Reading list already exists.")
				return err
			}
			if err != nil {
				return err
			}
		}
		forceMetadata := options.Force
		if existing && options.Mode == "full" && !forceMetadata {
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
		if err := importMetronReadingListWithOptions(ctx, db, client, covers, *list, options.Mode == "full" || options.Force, progress, options); err != nil {
			return err
		}
		return markMetronSynced(ctx, db, metronResourceReadingList, metronID, info)
	})
}

func startAllMetronReadingListsImport(parent context.Context, store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, options MetronImportOptions) MetronImportJob {
	options = resolveMetronImportOptions(options)
	return store.startWithContextAndOptions(parent, "readingLists", 0, options, "Fetching all reading lists from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		lists, err := client.ListReadingLists(ctx)
		if err != nil {
			return metronImportError(err)
		}
		progress(0, len(lists), "Importing all Metron reading lists...")
		failed := 0
		for i, summary := range lists {
			if err := ctx.Err(); err != nil {
				return err
			}
			list, err := client.GetReadingList(ctx, summary.ID)
			if err == nil {
				err = importMetronReadingListWithOptions(ctx, db, client, covers, *list, false, func(int, int, string) {}, options)
			}
			if err != nil {
				failed++
			}
			progress(i+1, len(lists), fmt.Sprintf("Imported %d of %d reading lists...", i+1-failed, len(lists)))
		}
		if failed > 0 {
			return fmt.Errorf("failed to import %d of %d reading lists", failed, len(lists))
		}
		return nil
	})
}

func startMetronArcImportWithOptions(parent context.Context, store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int, options MetronImportOptions) MetronImportJob {
	options = resolveMetronImportOptions(options)
	return store.startWithContextAndOptions(parent, "arc", metronID, options, "Importing arc and issues from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 0, "Checking existing imports...")
		if _, ok, err := existingArcIDByMetronID(ctx, db, metronID); err != nil || ok {
			if ok && options.Mode != "full" && !options.Force {
				progress(1, 1, "Arc already exists.")
				return err
			}
			if err != nil {
				return err
			}
		}

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
		if err := importMetronArcWithOptions(ctx, db, client, covers, *arc, options.Mode == "full" || options.Force, progress, options); err != nil {
			return err
		}
		return markMetronSynced(ctx, db, metronResourceArc, metronID, info)
	})
}

func startMetronSeriesImportWithOptions(parent context.Context, store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int, options MetronImportOptions) MetronImportJob {
	options = resolveMetronImportOptions(options)
	return store.startWithContextAndOptions(parent, "series", metronID, options, "Importing series from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 0, "Fetching series metadata from Metron...")
		metadata, info, err := fetchMetronSeries(ctx, db, client, metronID, options.Force)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceSeries, metronID); err != nil {
				return err
			}
		}

		progress(0, 0, "Fetching series issue list from Metron...")
		issues, err := client.GetSeriesIssues(ctx, metronID)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		_, err = importMetronSeriesWithProgressOptions(ctx, db, client, covers, issues, progress, options)
		if err != nil {
			return err
		}
		if metadata != nil {
			if err := updateImportedSeriesMetadata(ctx, db, *metadata); err != nil {
				return err
			}
		}
		return markMetronSynced(ctx, db, metronResourceSeries, metronID, info)
	})
}

func startLocalSeriesMetronImport(parent context.Context, store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, localID, metronID int) MetronImportJob {
	return store.startWithContextAndOptions(parent, "series", metronID, defaultMetronImportOptions(), "Importing series from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 0, "Fetching series metadata from Metron...")
		metadata, info, err := fetchMetronSeries(ctx, db, client, metronID, false)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceSeries, metronID); err != nil {
				return err
			}
		} else if err := updateSeriesMetronMetadata(ctx, db, localID, *metadata); err != nil {
			return err
		}

		progress(0, 0, "Fetching series issue list from Metron...")
		issues, err := client.GetSeriesIssues(ctx, metronID)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		_, err = importMetronSeriesWithProgress(ctx, db, client, covers, issues, progress)
		if err != nil {
			return err
		}
		return markMetronSynced(ctx, db, metronResourceSeries, metronID, info)
	})
}

func startMetronCharacterAppearancesImport(parent context.Context, store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int) MetronImportJob {
	return startMetronCharacterAppearancesImportWithOptions(parent, store, db, client, covers, metronID, defaultMetronImportOptions())
}

func startMetronCharacterAppearancesImportWithOptions(parent context.Context, store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int, options MetronImportOptions) MetronImportJob {
	options = resolveMetronImportOptions(options)
	return store.startWithContextAndOptions(parent, "character", metronID, options, "Importing character from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 0, "Preparing character appearance import...")
		return importMetronCharacterAppearancesWithProgressOptions(ctx, db, client, covers, metronID, progress, options)
	})
}
