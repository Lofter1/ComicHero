package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type SystemInfo struct {
	Version string `json:"version" doc:"Version of the running ComicHero build." example:"v1.5.1"`
}

type SystemInfoOutput struct {
	Body SystemInfo
}

func RegisterSystemRoutes(api huma.API, version string) {
	huma.Register(api, huma.Operation{
		OperationID: "getSystemInfo",
		Tags:        []string{tagSystem},
		Summary:     "Get system information",
		Description: "Returns public information about the running ComicHero build.",
		Method:      http.MethodGet,
		Path:        "/system",
	}, func(context.Context, *struct{}) (*SystemInfoOutput, error) {
		return &SystemInfoOutput{Body: SystemInfo{Version: version}}, nil
	})
}
