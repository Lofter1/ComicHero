package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

func TestCreateComicDownloadsRemoteCover(t *testing.T) {
	ctx := context.Background()
	db := newMetronImportTestDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("cover image"))
	}))
	defer server.Close()

	covers := NewCoverCache(t.TempDir(), "/covers")
	detail, err := createComic(ctx, db, covers, ComicPayload{
		Series:     "Series",
		Issue:      1,
		Publisher:  "Publisher",
		CoverImage: server.URL + "/cover",
	})
	if err != nil {
		t.Fatalf("createComic: %v", err)
	}

	if !strings.HasPrefix(detail.Body.CoverImage, "/covers/") {
		t.Fatalf("cover image = %q; want local cover URL", detail.Body.CoverImage)
	}
	if strings.HasPrefix(detail.Body.CoverImage, server.URL) {
		t.Fatalf("cover image kept remote URL: %q", detail.Body.CoverImage)
	}

	coverPath := filepath.Join(covers.dir, strings.TrimPrefix(detail.Body.CoverImage, "/covers/"))
	contents, err := os.ReadFile(coverPath)
	if err != nil {
		t.Fatalf("read cached cover: %v", err)
	}
	if string(contents) != "cover image" {
		t.Fatalf("cached cover = %q; want cover image", contents)
	}
}

func TestImportMetronComicDownloadsRemoteCover(t *testing.T) {
	ctx := context.Background()
	db := newMetronImportTestDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte("metron cover"))
	}))
	defer server.Close()

	covers := NewCoverCache(t.TempDir(), "/covers")
	detail, err := importMetronComic(ctx, db, nil, covers, metronIssueWithCover(server.URL+"/issue-cover.jpg"))
	if err != nil {
		t.Fatalf("importMetronComic: %v", err)
	}

	if !strings.HasPrefix(detail.Body.CoverImage, "/covers/") {
		t.Fatalf("cover image = %q; want local cover URL", detail.Body.CoverImage)
	}
	if !strings.HasSuffix(detail.Body.CoverImage, ".jpg") {
		t.Fatalf("cover image = %q; want jpg extension", detail.Body.CoverImage)
	}
}

func metronIssueWithCover(cover string) metron.Issue {
	return metron.Issue{
		ID:         101,
		Series:     "Series",
		SeriesYear: 2026,
		Issue:      1,
		Publisher:  "Publisher",
		CoverImage: cover,
	}
}
