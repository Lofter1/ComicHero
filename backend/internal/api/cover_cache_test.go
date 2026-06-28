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

func TestSyncMetronCharactersDownloadsRemoteImage(t *testing.T) {
	ctx := context.Background()
	db := newMetronImportTestDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("character image"))
	}))
	defer server.Close()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO comics (id, series, issue, publisher)
		VALUES (1, 'Series', 1, 'Publisher')
	`); err != nil {
		t.Fatalf("insert comic: %v", err)
	}

	covers := NewCoverCache(t.TempDir(), "/covers")
	err := syncMetronIssueCharacters(ctx, db, covers, 1, metron.Issue{
		Characters: []metron.MetronCharacter{{
			ID:    301,
			Name:  "Hero",
			Image: server.URL + "/hero",
		}},
	})
	if err != nil {
		t.Fatalf("syncMetronIssueCharacters: %v", err)
	}

	characters, err := listCharacters(ctx, db, &CharacterListInput{})
	if err != nil {
		t.Fatalf("listCharacters: %v", err)
	}
	if len(characters.Body) != 1 {
		t.Fatalf("characters = %d; want 1", len(characters.Body))
	}
	if !strings.HasPrefix(characters.Body[0].Image, "/covers/") {
		t.Fatalf("character image = %q; want local cover URL", characters.Body[0].Image)
	}
	if strings.HasPrefix(characters.Body[0].Image, server.URL) {
		t.Fatalf("character image kept remote URL: %q", characters.Body[0].Image)
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
