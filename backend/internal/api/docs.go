package api

import "github.com/danielgtaylor/huma/v2"

const (
	tagComics        = "Comics"
	tagDashboard     = "Dashboard"
	tagSeries        = "Series"
	tagCharacters    = "Characters"
	tagArcs          = "Arcs"
	tagReadingOrders = "Reading Orders"
	tagMetron        = "Metron"
	tagUsers         = "Users"
	tagStatistics    = "Statistics"
	tagSystem        = "System"
)

var (
	errsRead       = []int{400, 404, 500}
	errsWrite      = []int{400, 404, 422, 500}
	errsMetronRead = []int{400, 403, 429, 502}
	errsMetronSync = []int{400, 403, 404, 409, 429, 502, 500}
)

func DocsConfig() huma.Config {
	config := huma.DefaultConfig("ComicHero API", "0.1.0")
	config.OpenAPI.Info.Description = "ComicHero tracks comic reading orders, read progress, and metadata imported from Metron."
	config.OpenAPI.Servers = []*huma.Server{
		{URL: "/api", Description: "Current server"},
	}
	config.OpenAPI.Tags = []*huma.Tag{
		{Name: tagComics, Description: "Track comic metadata, read status, and reading-order membership."},
		{Name: tagDashboard, Description: "Summarize active reading queues and achievement highlights."},
		{Name: tagSeries, Description: "Browse local comic series, their read progress, and favorite state."},
		{Name: tagCharacters, Description: "Browse characters imported from Metron and their local comic appearances."},
		{Name: tagArcs, Description: "Manage story arcs and their ordered comic entries."},
		{Name: tagReadingOrders, Description: "Manage reading orders and their ordered comic entries."},
		{Name: tagMetron, Description: "Search, inspect, and import metadata from Metron."},
		{Name: tagUsers, Description: "Choose single-user or multi-user mode and manage login sessions."},
		{Name: tagStatistics, Description: "Summarize per-user reading progress and achievements."},
		{Name: tagSystem, Description: "Inspect public information about the running ComicHero build."},
	}
	return config
}
