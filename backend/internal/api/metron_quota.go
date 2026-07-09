package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/danielgtaylor/huma/v2"
)

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

func nullableMetronID(id int) any {
	if id <= 0 {
		return nil
	}
	return id
}

func nullableSeriesID(id int) any {
	if id <= 0 {
		return nil
	}
	return id
}
