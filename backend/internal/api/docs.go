package api

import "github.com/danielgtaylor/huma/v2"

const (
	tagComics        = "Comics"
	tagReadingOrders = "Reading Orders"
	tagMetron        = "Metron"
)

var (
	errsRead       = []int{400, 404, 500}
	errsWrite      = []int{400, 404, 422, 500}
	errsMetronRead = []int{400, 429, 502}
	errsMetronSync = []int{400, 404, 429, 502, 500}
)

func DocsConfig() huma.Config {
	config := huma.DefaultConfig("ComicHero API", "0.1.0")
	config.OpenAPI.Info.Description = "ComicHero tracks comic reading orders, read progress, and metadata imported from Metron."
	config.OpenAPI.Servers = []*huma.Server{
		{URL: "/", Description: "Current server"},
	}
	config.OpenAPI.Tags = []*huma.Tag{
		{Name: tagComics, Description: "Track comic metadata, read status, and reading-order membership."},
		{Name: tagReadingOrders, Description: "Manage reading orders and their ordered comic entries."},
		{Name: tagMetron, Description: "Search, inspect, and import metadata from Metron."},
	}
	return config
}
