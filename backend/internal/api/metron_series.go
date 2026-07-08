package api

import (
	"context"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/jmoiron/sqlx"
)

func importMetronSeries(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issues []metron.Issue) (*ComicListOutput, error) {
	return importMetronSeriesWithProgress(ctx, db, client, covers, issues, func(int, int, string) {})
}

func importMetronSeriesWithProgress(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issues []metron.Issue, progress func(int, int, string)) (*ComicListOutput, error) {
	return importMetronSeriesWithProgressOptions(ctx, db, client, covers, issues, progress, defaultMetronImportOptions())
}

func importMetronSeriesWithProgressOptions(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issues []metron.Issue, progress func(int, int, string), options MetronImportOptions) (*ComicListOutput, error) {
	options = resolveMetronImportOptions(options)
	comics := make([]Comic, 0, len(issues))
	total := len(issues)
	progress(0, total, "Importing series issues...")
	for i, issue := range issues {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if options.Mode != "full" && !options.Force && issue.ID > 0 {
			if id, ok, err := existingComicIDByMetronIssueID(ctx, db, issue.ID); err != nil {
				return nil, err
			} else if ok {
				comic, err := getComicRow(ctx, db, id)
				if err != nil {
					return nil, err
				}
				comics = append(comics, comic)
				progress(i+1, total, "Importing series issues...")
				continue
			}
		}

		if options.Mode != "full" && !options.Force {
			if id, ok, err := existingComicIDByMetronIssueMatch(ctx, db, issue); err != nil {
				return nil, err
			} else if ok {
				if issue.ID > 0 {
					if err := attachMetronIssueID(ctx, db, id, issue.ID); err != nil {
						return nil, err
					}
				}
				comic, err := getComicRow(ctx, db, id)
				if err != nil {
					return nil, err
				}
				comics = append(comics, comic)
				progress(i+1, total, "Importing series issues...")
				continue
			}
		}

		comic, err := importMetronComicSweep(ctx, db, client, covers, issue, options, true)
		if err != nil {
			return nil, err
		}
		comics = append(comics, comic.Body.Comic)
		progress(i+1, total, "Importing series issues...")
	}
	progress(total, total, "Series imported.")
	return &ComicListOutput{Body: comics}, nil
}
