package api

import "github.com/Lofter1/ComicHero/backend/internal/metron"

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
