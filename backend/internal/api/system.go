package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

type SystemInfo struct {
	Version         string `json:"version" doc:"Version of the running ComicHero build." example:"v1.5.1"`
	ShowVersion     bool   `json:"showVersion" doc:"Whether clients should display the running version."`
	LatestVersion   string `json:"latestVersion,omitempty" doc:"Latest published stable ComicHero version." example:"v1.6.0"`
	UpdateAvailable bool   `json:"updateAvailable" doc:"Whether a newer stable ComicHero release is available."`
	ReleaseURL      string `json:"releaseUrl,omitempty" doc:"Page for the latest stable ComicHero release."`
}

type SystemInfoOutput struct {
	Body SystemInfo
}

type LatestRelease struct {
	Version string
	URL     string
}

type ReleaseChecker interface {
	Latest(context.Context) (LatestRelease, error)
}

type GitHubReleaseChecker struct {
	client    *http.Client
	url       string
	cacheFor  time.Duration
	mu        sync.Mutex
	cached    LatestRelease
	expiresAt time.Time
}

func NewGitHubReleaseChecker() *GitHubReleaseChecker {
	return &GitHubReleaseChecker{
		client:   &http.Client{Timeout: 5 * time.Second},
		url:      "https://api.github.com/repos/Lofter1/ComicHero/releases/latest",
		cacheFor: 6 * time.Hour,
	}
}

func (checker *GitHubReleaseChecker) Latest(ctx context.Context) (LatestRelease, error) {
	checker.mu.Lock()
	defer checker.mu.Unlock()
	if time.Now().Before(checker.expiresAt) {
		return checker.cached, nil
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, checker.url, nil)
	if err != nil {
		return LatestRelease{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "ComicHero update checker")
	response, err := checker.client.Do(request)
	if err != nil {
		return LatestRelease{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return LatestRelease{}, fmt.Errorf("GitHub releases returned %s", response.Status)
	}
	var body struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return LatestRelease{}, err
	}
	checker.cached = LatestRelease{Version: body.TagName, URL: body.HTMLURL}
	checker.expiresAt = time.Now().Add(checker.cacheFor)
	return checker.cached, nil
}

func RegisterSystemRoutes(api huma.API, version string, showVersion bool, releaseChecker ReleaseChecker) {
	huma.Register(api, huma.Operation{
		OperationID: "getSystemInfo",
		Tags:        []string{tagSystem},
		Summary:     "Get system information",
		Description: "Returns public information about the running ComicHero build.",
		Method:      http.MethodGet,
		Path:        "/system",
	}, func(ctx context.Context, _ *struct{}) (*SystemInfoOutput, error) {
		info := SystemInfo{
			Version:     version,
			ShowVersion: showVersion,
		}
		if releaseChecker != nil {
			if latest, err := releaseChecker.Latest(ctx); err == nil {
				info.LatestVersion = latest.Version
				info.ReleaseURL = latest.URL
				info.UpdateAvailable = newerVersion(version, latest.Version)
			}
		}
		return &SystemInfoOutput{Body: info}, nil
	})
}

func newerVersion(current, latest string) bool {
	currentParts, currentOK := versionParts(current)
	latestParts, latestOK := versionParts(latest)
	if !currentOK || !latestOK {
		return false
	}
	for index := range currentParts {
		if latestParts[index] != currentParts[index] {
			return latestParts[index] > currentParts[index]
		}
	}
	return false
}

func versionParts(value string) ([3]int, bool) {
	var result [3]int
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	value = strings.SplitN(value, "-", 2)[0]
	parts := strings.Split(value, ".")
	if len(parts) != len(result) {
		return result, false
	}
	for index, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return result, false
		}
		result[index] = number
	}
	return result, true
}
