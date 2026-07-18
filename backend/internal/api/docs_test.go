package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	_ "modernc.org/sqlite"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

func TestDocsConfigAndRouteMetadata(t *testing.T) {
	router := chi.NewRouter()
	api := humachi.New(router, DocsConfig())
	RegisterComicRoutes(api, nil, nil)
	RegisterSeriesRoutes(api, nil, metron.New(metron.Config{}), nil, newMetronImportJobStore())
	RegisterCharacterRoutes(api, nil)
	RegisterCharacterCollectionRoutes(api, nil)
	RegisterReadingOrderRoutes(api, nil, nil)
	RegisterArcRoutes(api, nil)
	RegisterDashboardRoutes(api, nil)
	RegisterStatisticsRoutes(api, nil)
	RegisterSystemRoutes(api, "test")
	RegisterMetronRoutes(api, nil, metron.New(metron.Config{}), nil, newMetronImportJobStore())

	openAPI := api.OpenAPI()
	if openAPI.Info.Description == "" {
		t.Fatal("OpenAPI description is empty")
	}
	if len(openAPI.Tags) != 11 {
		t.Fatalf("len(tags) = %d; want 11", len(openAPI.Tags))
	}

	listComics := openAPI.Paths["/comics"].Get
	if len(listComics.Tags) != 1 || listComics.Tags[0] != tagComics {
		t.Fatalf("list comics tags = %#v; want Comics tag", listComics.Tags)
	}
	if _, ok := listComics.Responses["400"]; !ok {
		t.Fatal("list comics response docs missing 400 error")
	}

	accountStatistics := openAPI.Paths["/account/statistics"].Get
	if len(accountStatistics.Tags) != 1 || accountStatistics.Tags[0] != tagStatistics {
		t.Fatalf("account statistics tags = %#v; want Statistics tag", accountStatistics.Tags)
	}

	dashboard := openAPI.Paths["/dashboard"].Get
	if len(dashboard.Tags) != 1 || dashboard.Tags[0] != tagDashboard {
		t.Fatalf("dashboard tags = %#v; want Dashboard tag", dashboard.Tags)
	}

	listCharacters := openAPI.Paths["/characters"].Get
	if len(listCharacters.Tags) != 1 || listCharacters.Tags[0] != tagCharacters {
		t.Fatalf("list characters tags = %#v; want Characters tag", listCharacters.Tags)
	}

	listCollections := openAPI.Paths["/collections"].Get
	if len(listCollections.Tags) != 1 || listCollections.Tags[0] != tagCollections {
		t.Fatalf("list collections tags = %#v; want Collections tag", listCollections.Tags)
	}

	listSeries := openAPI.Paths["/series"].Get
	if len(listSeries.Tags) != 1 || listSeries.Tags[0] != tagSeries {
		t.Fatalf("list series tags = %#v; want Series tag", listSeries.Tags)
	}

	listArcs := openAPI.Paths["/arcs"].Get
	if len(listArcs.Tags) != 1 || listArcs.Tags[0] != tagArcs {
		t.Fatalf("list arcs tags = %#v; want Arcs tag", listArcs.Tags)
	}

	searchMetronArcs := openAPI.Paths["/metron/arcs"].Get
	if len(searchMetronArcs.Tags) != 1 || searchMetronArcs.Tags[0] != tagMetron {
		t.Fatalf("search Metron arcs tags = %#v; want Metron tag", searchMetronArcs.Tags)
	}

	importSeries := openAPI.Paths["/metron/series/{id}/import"].Post
	if len(importSeries.Tags) != 1 || importSeries.Tags[0] != tagMetron {
		t.Fatalf("import series tags = %#v; want Metron tag", importSeries.Tags)
	}
	if _, ok := importSeries.Responses["429"]; !ok {
		t.Fatal("import series response docs missing 429 error")
	}
}

func TestMountedDocsLoadMountedOpenAPISpec(t *testing.T) {
	router := chi.NewRouter()
	apiRouter := chi.NewRouter()
	router.Mount("/api", apiRouter)
	humachi.New(apiRouter, DocsConfig())

	docs := httptest.NewRecorder()
	router.ServeHTTP(docs, httptest.NewRequest(http.MethodGet, "/api/docs", nil))
	if docs.Code != http.StatusOK {
		t.Fatalf("docs status = %d; want 200", docs.Code)
	}
	if !strings.Contains(docs.Body.String(), `apiDescriptionUrl="/api/openapi.yaml"`) {
		t.Fatalf("docs body did not reference mounted OpenAPI spec: %s", docs.Body.String())
	}

	spec := httptest.NewRecorder()
	router.ServeHTTP(spec, httptest.NewRequest(http.MethodGet, "/api/openapi.yaml", nil))
	if spec.Code != http.StatusOK {
		t.Fatalf("spec status = %d; want 200", spec.Code)
	}
}
