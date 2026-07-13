package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

type staticReleaseChecker struct {
	release LatestRelease
}

func (checker staticReleaseChecker) Latest(context.Context) (LatestRelease, error) {
	return checker.release, nil
}

func TestSystemInfo(t *testing.T) {
	router := chi.NewRouter()
	api := humachi.New(router, DocsConfig())
	RegisterSystemRoutes(api, "v1.6.0", false, staticReleaseChecker{release: LatestRelease{
		Version: "v1.7.0",
		URL:     "https://github.com/Lofter1/ComicHero/releases/tag/v1.7.0",
	}})

	request := httptest.NewRequest(http.MethodGet, "/system", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", response.Code, http.StatusOK)
	}
	var body SystemInfo
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Version != "v1.6.0" || body.ShowVersion {
		t.Fatalf("body = %#v; want version v1.6.0 with display disabled", body)
	}
	if !body.UpdateAvailable || body.LatestVersion != "v1.7.0" || body.ReleaseURL == "" {
		t.Fatalf("body = %#v; want v1.7.0 update information", body)
	}
}

func TestNewerVersion(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{current: "v1.5.1", latest: "v1.6.0", want: true},
		{current: "1.6.0", latest: "v1.6.0", want: false},
		{current: "v2.0.0", latest: "v1.9.9", want: false},
		{current: "dev", latest: "v1.6.0", want: false},
	}
	for _, test := range tests {
		if got := newerVersion(test.current, test.latest); got != test.want {
			t.Errorf("newerVersion(%q, %q) = %v; want %v", test.current, test.latest, got, test.want)
		}
	}
}

func TestGitHubReleaseCheckerCachesLatestRelease(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"tag_name":"v1.7.0","html_url":"https://example.com/v1.7.0"}`))
	}))
	defer server.Close()

	checker := &GitHubReleaseChecker{
		client:   server.Client(),
		url:      server.URL,
		cacheFor: time.Hour,
	}
	for range 2 {
		release, err := checker.Latest(context.Background())
		if err != nil {
			t.Fatalf("latest release: %v", err)
		}
		if release.Version != "v1.7.0" || release.URL != "https://example.com/v1.7.0" {
			t.Fatalf("release = %#v; want v1.7.0", release)
		}
	}
	if requests != 1 {
		t.Fatalf("requests = %d; want 1 cached request", requests)
	}
}
