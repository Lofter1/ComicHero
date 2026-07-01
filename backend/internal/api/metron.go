package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"
	"github.com/jmoiron/sqlx"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

type MetronIssueListInput struct {
	Query  string `query:"q"      doc:"Search text for Metron issues. Used as series-name search when series is empty." example:"Batman"`
	Series string `query:"series" doc:"Metron series-name filter." example:"Batman"`
	Issue  string `query:"issue"  doc:"Issue-number filter." example:"6.LR"`
}

type MetronIDInput struct {
	ID int `path:"id" doc:"Metron resource identifier." minimum:"1" example:"123456"`
}

type MetronImportOptions struct {
	Mode     string   `json:"mode,omitempty"     doc:"Import depth preset. Use quick for the base Metron endpoints or full for detail expansion." enum:"quick,full" example:"quick"`
	FullData []string `json:"fullData,omitempty" doc:"Full-import data areas to pull. Supported values are comics, series, arcs, and characters. Characters, arcs, and series imply comic issue details." example:"comics"`
	Force    bool     `json:"force,omitempty"    doc:"Bypass Metron conditional requests and download fresh metadata even when local sync state is current." example:"false"`
}

type MetronImportInput struct {
	ID   int `path:"id" doc:"Metron resource identifier." minimum:"1" example:"123456"`
	Body MetronImportOptions
}

type MetronReadingListInput struct {
	Query string `query:"q" doc:"Search text for Metron reading lists." example:"Court of Owls"`
}

type MetronArcInput struct {
	Query string `query:"q" doc:"Search text for Metron story arcs." example:"Zero Year"`
}

type MetronSeriesInput struct {
	Query     string `query:"q" doc:"Search text for Metron series." example:"Batman"`
	YearBegan int    `query:"year_began" doc:"Filter series by starting year." minimum:"1" example:"2018"`
	Volume    int    `query:"volume" doc:"Filter series by volume number." minimum:"1" example:"1"`
}

type MetronCharacterInput struct {
	Query string `query:"q" doc:"Search text for Metron characters." example:"Batman"`
}

type MetronIssueListOutput struct {
	MetronRateLimitHeaders
	Body []metron.Issue
}

type MetronIssueOutput struct {
	MetronRateLimitHeaders
	Body metron.Issue
}

type MetronReadingListOutput struct {
	MetronRateLimitHeaders
	Body []metron.ReadingList
}

type MetronReadingListDetailOutput struct {
	MetronRateLimitHeaders
	Body metron.ReadingList
}

type MetronArcListOutput struct {
	MetronRateLimitHeaders
	Body []metron.MetronArc
}

type MetronArcDetailOutput struct {
	MetronRateLimitHeaders
	Body metron.MetronArc
}

type MetronSeriesListOutput struct {
	MetronRateLimitHeaders
	Body []metron.Series
}

type MetronCharacterListOutput struct {
	MetronRateLimitHeaders
	Body []metron.MetronCharacter
}

type MetronRateLimitHeaders struct {
	BurstLimit         string `header:"X-RateLimit-Burst-Limit"         doc:"Metron burst-rate request limit, forwarded from the latest Metron response."`
	BurstRemaining     string `header:"X-RateLimit-Burst-Remaining"     doc:"Remaining Metron burst-rate requests, forwarded from the latest Metron response."`
	BurstReset         string `header:"X-RateLimit-Burst-Reset"         doc:"Unix timestamp when the Metron burst-rate window resets."`
	SustainedLimit     string `header:"X-RateLimit-Sustained-Limit"     doc:"Metron sustained-rate request limit, forwarded from the latest Metron response."`
	SustainedRemaining string `header:"X-RateLimit-Sustained-Remaining" doc:"Remaining Metron sustained-rate requests, forwarded from the latest Metron response."`
	SustainedReset     string `header:"X-RateLimit-Sustained-Reset"     doc:"Unix timestamp when the Metron sustained-rate window resets."`
}

type MetronQuota struct {
	BurstLimit         int   `json:"burstLimit"         doc:"Metron burst-rate request limit." example:"10"`
	BurstRemaining     int   `json:"burstRemaining"     doc:"Remaining Metron burst-rate requests." example:"4"`
	BurstUsed          int   `json:"burstUsed"          doc:"Used Metron burst-rate requests in the current window." example:"6"`
	BurstReset         int64 `json:"burstReset"         doc:"Unix timestamp when the burst-rate window resets." example:"1782468300"`
	SustainedLimit     int   `json:"sustainedLimit"     doc:"Metron sustained-rate request limit." example:"100"`
	SustainedRemaining int   `json:"sustainedRemaining" doc:"Remaining Metron sustained-rate requests." example:"75"`
	SustainedUsed      int   `json:"sustainedUsed"      doc:"Used Metron sustained-rate requests in the current window." example:"25"`
	SustainedReset     int64 `json:"sustainedReset"     doc:"Unix timestamp when the sustained-rate window resets." example:"1782470000"`
	Known              bool  `json:"known"              doc:"Whether Metron has returned quota headers during this server run." example:"true"`
}

type MetronQuotaOutput struct {
	MetronRateLimitHeaders
	Body MetronQuota
}

type MetronRequestLogOutput struct {
	Body []metron.RequestLogEntry
}

func RegisterMetronRoutes(api huma.API, db *sqlx.DB, client *metron.Client, covers *CoverCache, importJobs *metronImportJobStore) {
	huma.Register(api, huma.Operation{
		OperationID: "searchMetronComics",
		Tags:        []string{tagMetron},
		Summary:     "Search Metron comics",
		Description: "Searches Metron for comic issue metadata. When series is omitted, q is sent as the Metron series-name search.",
		Method:      http.MethodGet,
		Path:        "/metron/comics",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronIssueListInput) (*MetronIssueListOutput, error) {
		issues, err := client.SearchIssues(ctx, input.Query, input.Series, input.Issue)
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronIssueListOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: issues}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getMetronComic",
		Tags:        []string{tagMetron},
		Summary:     "Get Metron comic",
		Description: "Gets a Metron comic issue by ID.",
		Method:      http.MethodGet,
		Path:        "/metron/comics/{id}",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronIDInput) (*MetronIssueOutput, error) {
		issue, err := client.GetIssue(ctx, input.ID)
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronIssueOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: *issue}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "importMetronComic",
		Tags:          []string{tagMetron},
		Summary:       "Import Metron comic",
		Description:   "Starts a background job that imports a Metron comic issue for use in reading orders. If the issue is already imported, the job finishes without calling Metron again.",
		Method:        http.MethodPost,
		Path:          "/metron/comics/{id}/import",
		DefaultStatus: http.StatusAccepted,
		Errors:        errsMetronSync,
	}, func(ctx context.Context, input *MetronImportInput) (*MetronImportJobOutput, error) {
		job := startMetronComicImportWithOptions(importJobs, db, client, covers, input.ID, input.Body)
		return &MetronImportJobOutput{Body: job}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateComicFromMetron",
		Tags:        []string{tagMetron},
		Summary:     "Update comic from Metron",
		Description: "Updates an existing comic's metadata from a Metron issue while preserving local read status.",
		Method:      http.MethodPatch,
		Path:        "/comics/{id}/metron",
		Errors:      errsMetronSync,
	}, func(ctx context.Context, input *UpdateComicFromMetronInput) (*ComicDetailOutput, error) {
		issue, info, err := fetchMetronIssue(ctx, db, client, input.Body.MetronIssueID, input.Body.Force)
		if err != nil {
			return nil, metronAPIError(err)
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceIssue, input.Body.MetronIssueID); err != nil {
				return nil, err
			}
			output, err := getComic(ctx, db, input.ID)
			if err != nil {
				return nil, err
			}
			return withMetronRateLimit(output, client.CurrentRateLimit()), nil
		}
		output, err := updateComicFromMetron(ctx, db, client, covers, input.ID, *issue)
		if err != nil {
			return nil, err
		}
		if err := markMetronSynced(ctx, db, metronResourceIssue, input.Body.MetronIssueID, info); err != nil {
			return nil, err
		}
		return withMetronRateLimit(output, client.CurrentRateLimit()), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "searchMetronReadingLists",
		Tags:        []string{tagMetron},
		Summary:     "Search Metron reading lists",
		Description: "Searches Metron for reading lists.",
		Method:      http.MethodGet,
		Path:        "/metron/readingLists",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronReadingListInput) (*MetronReadingListOutput, error) {
		lists, err := client.SearchReadingLists(ctx, input.Query)
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronReadingListOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: lists}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getMetronReadingList",
		Tags:        []string{tagMetron},
		Summary:     "Get Metron reading list",
		Description: "Gets a Metron reading list by ID, including its issue entries when Metron returns them.",
		Method:      http.MethodGet,
		Path:        "/metron/readingLists/{id}",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronIDInput) (*MetronReadingListDetailOutput, error) {
		list, err := client.GetReadingList(ctx, input.ID)
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronReadingListDetailOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: *list}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "importMetronReadingList",
		Tags:          []string{tagMetron},
		Summary:       "Import Metron reading list",
		Description:   "Starts a background job that imports a Metron reading list as a local reading order and imports its issues as local comics. If the reading list is already imported, the job finishes without calling Metron again.",
		Method:        http.MethodPost,
		Path:          "/metron/readingLists/{id}/import",
		DefaultStatus: http.StatusAccepted,
		Errors:        errsMetronSync,
	}, func(ctx context.Context, input *MetronImportInput) (*MetronImportJobOutput, error) {
		job := startMetronReadingListImportWithOptions(importJobs, db, client, covers, input.ID, input.Body)
		return &MetronImportJobOutput{Body: job}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "searchMetronArcs",
		Tags:        []string{tagMetron},
		Summary:     "Search Metron arcs",
		Description: "Searches Metron for story arcs.",
		Method:      http.MethodGet,
		Path:        "/metron/arcs",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronArcInput) (*MetronArcListOutput, error) {
		arcs, err := client.SearchArcs(ctx, input.Query)
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronArcListOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: arcs}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getMetronArc",
		Tags:        []string{tagMetron},
		Summary:     "Get Metron arc",
		Description: "Gets a Metron story arc by ID, including its issue entries.",
		Method:      http.MethodGet,
		Path:        "/metron/arcs/{id}",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronIDInput) (*MetronArcDetailOutput, error) {
		arc, err := client.GetArc(ctx, input.ID)
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronArcDetailOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: *arc}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "importMetronArc",
		Tags:          []string{tagMetron, tagArcs},
		Summary:       "Import Metron arc",
		Description:   "Starts a background job that imports a Metron story arc as a local arc and imports or reuses its issues as local comics. If the arc is already imported, the job finishes without calling Metron again.",
		Method:        http.MethodPost,
		Path:          "/metron/arcs/{id}/import",
		DefaultStatus: http.StatusAccepted,
		Errors:        errsMetronSync,
	}, func(ctx context.Context, input *MetronImportInput) (*MetronImportJobOutput, error) {
		job := startMetronArcImportWithOptions(importJobs, db, client, covers, input.ID, input.Body)
		return &MetronImportJobOutput{Body: job}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "searchMetronSeries",
		Tags:        []string{tagMetron},
		Summary:     "Search Metron series",
		Description: "Searches Metron for comic series.",
		Method:      http.MethodGet,
		Path:        "/metron/series",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronSeriesInput) (*MetronSeriesListOutput, error) {
		series, err := client.SearchSeries(ctx, metron.SeriesSearchOptions{
			Query:     input.Query,
			YearBegan: input.YearBegan,
			Volume:    input.Volume,
		})
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronSeriesListOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: series}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "searchMetronCharacters",
		Tags:        []string{tagMetron},
		Summary:     "Search Metron characters",
		Description: "Searches Metron for characters by name.",
		Method:      http.MethodGet,
		Path:        "/metron/characters",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronCharacterInput) (*MetronCharacterListOutput, error) {
		characters, err := client.SearchCharacters(ctx, input.Query)
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronCharacterListOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: characters}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "importMetronCharacterAppearances",
		Tags:          []string{tagMetron, tagCharacters},
		Summary:       "Import Metron character appearances",
		Description:   "Starts a background job that imports or reuses a Metron character locally, fetches the character's Metron issue list, imports or reuses those issues, and links them as local appearances.",
		Method:        http.MethodPost,
		Path:          "/metron/characters/{id}/import",
		DefaultStatus: http.StatusAccepted,
		Errors:        errsMetronSync,
	}, func(ctx context.Context, input *MetronImportInput) (*MetronImportJobOutput, error) {
		job := startMetronCharacterAppearancesImportWithOptions(importJobs, db, client, covers, input.ID, input.Body)
		return &MetronImportJobOutput{Body: job}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "importMetronSeries",
		Tags:          []string{tagMetron},
		Summary:       "Import Metron series",
		Description:   "Starts a background job that imports every missing issue in a Metron series for use in reading orders. Already imported issues are skipped before detail calls are made.",
		Method:        http.MethodPost,
		Path:          "/metron/series/{id}/import",
		DefaultStatus: http.StatusAccepted,
		Errors:        errsMetronSync,
	}, func(ctx context.Context, input *MetronImportInput) (*MetronImportJobOutput, error) {
		job := startMetronSeriesImportWithOptions(importJobs, db, client, covers, input.ID, input.Body)
		return &MetronImportJobOutput{Body: job}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getMetronQuota",
		Tags:        []string{tagMetron},
		Summary:     "Get Metron quota",
		Description: "Returns the latest Metron rate-limit quota known to this server.",
		Method:      http.MethodGet,
		Path:        "/metron/quota",
		Errors:      errsRead,
	}, func(ctx context.Context, input *struct{}) (*MetronQuotaOutput, error) {
		rateLimit := client.CurrentRateLimit()
		return &MetronQuotaOutput{
			MetronRateLimitHeaders: metronRateLimitHeaders(rateLimit),
			Body:                   metronQuotaFromRateLimit(rateLimit),
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listMetronImportJobs",
		Tags:        []string{tagMetron},
		Summary:     "List Metron import jobs",
		Description: "Returns background Metron import jobs so the web app can reconnect after a reload.",
		Method:      http.MethodGet,
		Path:        "/metron/imports",
		Errors:      errsRead,
	}, func(ctx context.Context, input *struct{}) (*MetronImportJobListOutput, error) {
		return listMetronImportJobs(importJobs), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listMetronRequests",
		Tags:        []string{tagMetron},
		Summary:     "List recent Metron requests",
		Description: "Returns recent outbound Metron API calls recorded by this server, including path, query, status, duration, and conditional-request state.",
		Method:      http.MethodGet,
		Path:        "/metron/requests",
		Errors:      errsRead,
	}, func(ctx context.Context, input *struct{}) (*MetronRequestLogOutput, error) {
		return &MetronRequestLogOutput{Body: client.RecentRequests()}, nil
	})

	sse.Register(api, huma.Operation{
		OperationID: "streamMetronImportJobs",
		Tags:        []string{tagMetron},
		Summary:     "Stream Metron import jobs",
		Description: "Streams background Metron import job updates so the web app can reconnect after a reload without polling.",
		Method:      http.MethodGet,
		Path:        "/metron/imports/events",
		Errors:      errsRead,
	}, map[string]any{
		"job": MetronImportJobEvent{},
	}, func(ctx context.Context, input *struct{}, send sse.Sender) {
		streamMetronImportJobs(ctx, importJobs, func(event MetronImportJobEvent) error {
			return send.Data(event)
		})
	})

	huma.Register(api, huma.Operation{
		OperationID: "getMetronImportJob",
		Tags:        []string{tagMetron},
		Summary:     "Get Metron import job",
		Description: "Returns the current status of a background Metron import job.",
		Method:      http.MethodGet,
		Path:        "/metron/imports/{id}",
		Errors:      errsRead,
	}, func(ctx context.Context, input *MetronImportJobInput) (*MetronImportJobOutput, error) {
		return getMetronImportJob(importJobs, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "dismissMetronImportJob",
		Tags:        []string{tagMetron},
		Summary:     "Dismiss Metron import job",
		Description: "Removes a finished Metron import job from the monitor.",
		Method:      http.MethodDelete,
		Path:        "/metron/imports/{id}",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *MetronImportJobInput) (*struct{}, error) {
		return deleteMetronImportJob(importJobs, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "cancelMetronImportJob",
		Tags:        []string{tagMetron},
		Summary:     "Cancel Metron import job",
		Description: "Requests cancellation for a queued or running background Metron import job.",
		Method:      http.MethodPost,
		Path:        "/metron/imports/{id}/cancel",
		Errors:      errsRead,
	}, func(ctx context.Context, input *MetronImportJobInput) (*MetronImportJobOutput, error) {
		return cancelMetronImportJob(importJobs, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "continueMetronImportJob",
		Tags:          []string{tagMetron},
		Summary:       "Continue Metron import job",
		Description:   "Starts a new background import for the same Metron resource as a canceled import job.",
		Method:        http.MethodPost,
		Path:          "/metron/imports/{id}/continue",
		DefaultStatus: http.StatusAccepted,
		Errors:        errsMetronSync,
	}, func(ctx context.Context, input *MetronImportJobInput) (*MetronImportJobOutput, error) {
		return continueMetronImportJob(importJobs, db, client, covers, input.ID)
	})
}

func metronAPIError(err error) error {
	var rateLimitErr *metron.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return huma.NewError(http.StatusTooManyRequests, rateLimitErr.Error())
	}
	return huma.Error502BadGateway(err.Error())
}

func metronRateLimitHeaders(rateLimit metron.RateLimit) MetronRateLimitHeaders {
	if rateLimit.Empty() {
		return MetronRateLimitHeaders{}
	}
	return MetronRateLimitHeaders{
		BurstLimit:         strconv.Itoa(rateLimit.BurstLimit),
		BurstRemaining:     strconv.Itoa(rateLimit.BurstRemaining),
		BurstReset:         strconv.FormatInt(rateLimit.BurstReset, 10),
		SustainedLimit:     strconv.Itoa(rateLimit.SustainedLimit),
		SustainedRemaining: strconv.Itoa(rateLimit.SustainedRemaining),
		SustainedReset:     strconv.FormatInt(rateLimit.SustainedReset, 10),
	}
}

func metronQuotaFromRateLimit(rateLimit metron.RateLimit) MetronQuota {
	quota := MetronQuota{
		BurstLimit:         rateLimit.BurstLimit,
		BurstRemaining:     rateLimit.BurstRemaining,
		BurstReset:         rateLimit.BurstReset,
		SustainedLimit:     rateLimit.SustainedLimit,
		SustainedRemaining: rateLimit.SustainedRemaining,
		SustainedReset:     rateLimit.SustainedReset,
		Known:              !rateLimit.Empty(),
	}
	if quota.BurstLimit >= quota.BurstRemaining {
		quota.BurstUsed = quota.BurstLimit - quota.BurstRemaining
	}
	if quota.SustainedLimit >= quota.SustainedRemaining {
		quota.SustainedUsed = quota.SustainedLimit - quota.SustainedRemaining
	}
	return quota
}

func withMetronRateLimit[T interface {
	*ComicDetailOutput | *ReadingOrderDetailOutput | *ComicListOutput | *CharacterDetailOutput | *ArcDetailOutput
}](output T, rateLimit metron.RateLimit) T {
	headers := metronRateLimitHeaders(rateLimit)
	switch typed := any(output).(type) {
	case *ComicDetailOutput:
		typed.MetronRateLimitHeaders = headers
	case *ReadingOrderDetailOutput:
		typed.MetronRateLimitHeaders = headers
	case *ComicListOutput:
		typed.MetronRateLimitHeaders = headers
	case *CharacterDetailOutput:
		typed.MetronRateLimitHeaders = headers
	case *ArcDetailOutput:
		typed.MetronRateLimitHeaders = headers
	}
	return output
}

func comicPayloadFromMetronIssue(issue metron.Issue) ComicPayload {
	return ComicPayload{
		Series:      issue.Series,
		SeriesYear:  issue.SeriesYear,
		Issue:       issue.Issue,
		Publisher:   issue.Publisher,
		CoverDate:   issue.CoverDate,
		CoverImage:  issue.CoverImage,
		Description: issue.Description,
	}
}

func importMetronComic(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issue metron.Issue) (*ComicDetailOutput, error) {
	return importMetronComicWithOptions(ctx, db, client, covers, issue, defaultMetronImportOptions())
}

func importMetronComicWithOptions(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issue metron.Issue, options MetronImportOptions) (*ComicDetailOutput, error) {
	options = resolveMetronImportOptions(options)
	if issue.ID > 0 {
		if id, ok, err := existingComicIDByMetronIssueID(ctx, db, issue.ID); err != nil {
			return nil, err
		} else if ok {
			if options.Force {
				return updateComicFromMetron(ctx, db, client, covers, id, issue)
			}
			if err := syncMetronIssueArcsWithOptions(ctx, db, client, id, issue, options); err != nil {
				return nil, err
			}
			if options.includesCharacters() {
				if err := syncMetronIssueCharactersWithOptions(ctx, db, client, covers, id, issue, options); err != nil {
					return nil, err
				}
			}
			return getComic(ctx, db, id)
		}
	}

	if id, ok, err := existingComicIDByMetronIssueMatch(ctx, db, issue); err != nil {
		return nil, err
	} else if ok {
		if issue.ID > 0 {
			if err := attachMetronIssueID(ctx, db, id, issue.ID); err != nil {
				return nil, err
			}
		}
		if options.Force {
			return updateComicFromMetron(ctx, db, client, covers, id, issue)
		}
		if err := syncMetronIssueArcsWithOptions(ctx, db, client, id, issue, options); err != nil {
			return nil, err
		}
		if options.includesCharacters() {
			if err := syncMetronIssueCharactersWithOptions(ctx, db, client, covers, id, issue, options); err != nil {
				return nil, err
			}
		}
		return getComic(ctx, db, id)
	}

	return createMetronComicWithOptions(ctx, db, client, covers, issue, options)
}

func importMetronComicSweep(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issue metron.Issue, options MetronImportOptions, fetchIssueDetail bool) (*ComicDetailOutput, error) {
	options = resolveMetronImportOptions(options)
	var issueInfo metron.FetchInfo
	if options.needsIssueDetail() && fetchIssueDetail && client != nil && issue.ID > 0 {
		if !options.Force {
			if id, ok, err := existingComicIDByMetronIssueID(ctx, db, issue.ID); err != nil {
				return nil, err
			} else if ok {
				complete, err := comicHasRequestedMetronData(ctx, db, id, options)
				if err != nil {
					return nil, err
				}
				if complete {
					return getComic(ctx, db, id)
				}
			}
		}
		forceIssue := options.Force
		if id, ok, err := existingComicIDByMetronIssueID(ctx, db, issue.ID); err != nil {
			return nil, err
		} else if ok {
			complete, err := comicHasRequestedMetronData(ctx, db, id, options)
			if err != nil {
				return nil, err
			}
			forceIssue = forceIssue || !complete
		}
		detail, info, err := fetchMetronIssue(ctx, db, client, issue.ID, forceIssue)
		if err != nil {
			if isContextCanceledError(err) {
				return nil, err
			}
			return nil, huma.Error502BadGateway(err.Error())
		}
		issueInfo = info
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceIssue, issue.ID); err != nil {
				return nil, err
			}
			if id, ok, err := existingComicIDByMetronIssueID(ctx, db, issue.ID); err != nil {
				return nil, err
			} else if ok {
				return getComic(ctx, db, id)
			}
			detail, issueInfo, err = fetchMetronIssue(ctx, db, client, issue.ID, true)
			if err != nil {
				return nil, huma.Error502BadGateway(err.Error())
			}
			issue = *detail
		} else {
			issue = *detail
		}
	}

	comic, err := importMetronComicWithOptions(ctx, db, client, covers, issue, options)
	if err != nil {
		return nil, err
	}
	if options.includesComics() && fetchIssueDetail && issue.ID > 0 {
		if err := markMetronSynced(ctx, db, metronResourceIssue, issue.ID, issueInfo); err != nil {
			return nil, err
		}
	}
	if options.includesSeries() && client != nil && issue.SeriesID > 0 {
		metadata, info, err := fetchMetronSeries(ctx, db, client, issue.SeriesID, options.Force)
		if err != nil {
			if isContextCanceledError(err) {
				return nil, err
			}
			return nil, huma.Error502BadGateway(err.Error())
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceSeries, issue.SeriesID); err != nil {
				return nil, err
			}
		} else {
			if err := updateImportedSeriesMetadata(ctx, db, *metadata); err != nil {
				return nil, err
			}
			if err := markMetronSynced(ctx, db, metronResourceSeries, issue.SeriesID, info); err != nil {
				return nil, err
			}
		}
	}
	return comic, nil
}

func existingComicIDByMetronIssueID(ctx context.Context, db *sqlx.DB, metronID int) (int, bool, error) {
	var id int
	if err := db.GetContext(ctx, &id, `
		SELECT id FROM comics WHERE metron_issue_id = ?
	`, metronID); err != nil {
		if err != sql.ErrNoRows {
			return 0, false, huma.Error500InternalServerError("failed to check imported comic")
		}
		return 0, false, nil
	}
	return id, true, nil
}

func existingComicIDByMetronIssueMatch(ctx context.Context, db *sqlx.DB, issue metron.Issue) (int, bool, error) {
	var id int
	if err := db.GetContext(ctx, &id, `
		SELECT id FROM comics
		WHERE metron_issue_id IS NULL
			AND series = ?
			AND series_year = ?
			AND issue = ?
			AND publisher = ?
		ORDER BY id
		LIMIT 1
	`, issue.Series, issue.SeriesYear, issue.Issue, issue.Publisher); err != nil {
		if err != sql.ErrNoRows {
			return 0, false, huma.Error500InternalServerError("failed to check matching comic")
		}
		return 0, false, nil
	}
	return id, true, nil
}

func comicHasRequestedMetronData(ctx context.Context, db *sqlx.DB, comicID int, options MetronImportOptions) (bool, error) {
	if !options.needsIssueDetail() {
		return true, nil
	}
	var comic Comic
	if err := db.GetContext(ctx, &comic, `SELECT * FROM comics WHERE id = ?`, comicID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, huma.Error500InternalServerError("failed to check imported comic")
	}
	if options.includesComics() && (comic.Description == "" || comic.CoverImage == "") {
		return false, nil
	}
	if options.includesSeries() {
		var count int
		if err := db.GetContext(ctx, &count, `
			SELECT COUNT(*) FROM series
			WHERE name = ? AND series_year = ? AND (metron_series_id IS NOT NULL OR description <> '' OR issue_count > 0)
		`, comic.Series, comic.SeriesYear); err != nil {
			return false, huma.Error500InternalServerError("failed to check imported series metadata")
		}
		if count == 0 {
			return false, nil
		}
	}
	if options.includesArcs() {
		var count int
		if err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM arc_comics WHERE comic_id = ?`, comicID); err != nil {
			return false, huma.Error500InternalServerError("failed to check imported arcs")
		}
		if count == 0 {
			return false, nil
		}
	}
	if options.includesCharacters() {
		var count int
		if err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM comic_characters WHERE comic_id = ?`, comicID); err != nil {
			return false, huma.Error500InternalServerError("failed to check imported characters")
		}
		if count == 0 {
			return false, nil
		}
	}
	return true, nil
}

func attachMetronIssueID(ctx context.Context, db *sqlx.DB, comicID, metronID int) error {
	if _, err := db.ExecContext(ctx, `
		UPDATE comics SET metron_issue_id = ? WHERE id = ? AND metron_issue_id IS NULL
	`, metronID, comicID); err != nil {
		return huma.Error500InternalServerError("failed to link comic to Metron")
	}
	return nil
}

func createMetronComicWithOptions(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issue metron.Issue, options MetronImportOptions) (*ComicDetailOutput, error) {
	options = resolveMetronImportOptions(options)
	payload := comicPayloadFromMetronIssue(issue)
	var err error
	payload.CoverImage, err = localCoverURL(ctx, covers, payload.CoverImage)
	if err != nil {
		return nil, err
	}

	result, err := db.ExecContext(ctx, `
		INSERT INTO comics (series, series_year, issue, publisher, cover_date, cover_image, description, read, metron_issue_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, payload.Series,
		payload.SeriesYear,
		payload.Issue,
		payload.Publisher,
		payload.CoverDate,
		payload.CoverImage,
		payload.Description,
		payload.Read,
		nullableMetronID(issue.ID),
	)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to import Metron comic")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get imported comic id")
	}
	if err := ensureSeriesRow(ctx, db, payload.Series, payload.SeriesYear); err != nil {
		return nil, err
	}
	if err := syncMetronIssueArcsWithOptions(ctx, db, client, int(id), issue, options); err != nil {
		return nil, err
	}
	if options.includesCharacters() {
		if err := syncMetronIssueCharactersWithOptions(ctx, db, client, covers, int(id), issue, options); err != nil {
			return nil, err
		}
	}
	return getComic(ctx, db, int(id))
}

func updateComicFromMetron(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, comicID int, issue metron.Issue) (*ComicDetailOutput, error) {
	payload := comicPayloadFromMetronIssue(issue)
	var err error
	payload.CoverImage, err = localCoverURL(ctx, covers, payload.CoverImage)
	if err != nil {
		return nil, err
	}

	result, err := db.ExecContext(ctx, `
		UPDATE comics
		SET series = ?, series_year = ?, issue = ?, publisher = ?, cover_date = ?, cover_image = ?, description = ?, metron_issue_id = ?
		WHERE id = ?
	`, payload.Series,
		payload.SeriesYear,
		payload.Issue,
		payload.Publisher,
		payload.CoverDate,
		payload.CoverImage,
		payload.Description,
		nullableMetronID(issue.ID),
		comicID,
	)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to update comic from Metron")
	}
	if err := requireRowsAffected(result, "comic not found"); err != nil {
		return nil, err
	}
	if err := ensureSeriesRow(ctx, db, payload.Series, payload.SeriesYear); err != nil {
		return nil, err
	}

	if err := syncMetronIssueArcsWithOptions(ctx, db, client, comicID, issue, MetronImportOptions{Mode: "full"}); err != nil {
		return nil, err
	}
	if err := syncMetronIssueCharacters(ctx, db, covers, comicID, issue); err != nil {
		return nil, err
	}
	return getComic(ctx, db, comicID)
}

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

	return setReadingOrderComics(ctx, db, input)
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
		if _, err := setReadingOrderComics(ctx, db, input); err != nil {
			return err
		}
		progress(i+1, total, "Importing reading-list issues...")
	}

	if _, err := setReadingOrderComics(ctx, db, input); err != nil {
		return err
	}
	progress(total, total, "Reading list imported.")
	return nil
}

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

func createMetronReadingOrder(ctx context.Context, db *sqlx.DB, covers *CoverCache, list metron.ReadingList) (*CreateReadingOrderOutput, error) {
	image, err := localCoverURL(ctx, covers, list.Image)
	if err != nil {
		return nil, err
	}

	result, err := db.ExecContext(ctx, `
		INSERT INTO reading_orders (name, description, image, favorite, metron_reading_list_id)
		VALUES (?, ?, ?, ?, ?)
	`, list.Name, list.Description, image, false, nullableMetronID(list.ID))
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to import Metron reading list")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get imported reading order id")
	}

	var ro ReadingOrder
	if err := db.GetContext(ctx, &ro, `
		SELECT * FROM reading_orders WHERE id = ?
	`, id); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch imported reading order")
	}

	return &CreateReadingOrderOutput{Body: ro}, nil
}

func updateMetronReadingOrderMetadata(ctx context.Context, db *sqlx.DB, covers *CoverCache, id int, list metron.ReadingList) error {
	image, err := localCoverURL(ctx, covers, list.Image)
	if err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx, `
		UPDATE reading_orders
		SET name = COALESCE(NULLIF(?, ''), name),
			description = COALESCE(NULLIF(?, ''), description),
			image = COALESCE(NULLIF(?, ''), image),
			metron_reading_list_id = COALESCE(?, metron_reading_list_id)
		WHERE id = ?
	`, list.Name, list.Description, image, nullableMetronID(list.ID), id); err != nil {
		return huma.Error500InternalServerError("failed to update Metron reading list")
	}
	return nil
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

func nullableMetronID(id int) any {
	if id <= 0 {
		return nil
	}
	return id
}
