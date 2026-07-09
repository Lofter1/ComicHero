package api

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

func TestCreateComicDownloadsRemoteCover(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(testPNG(t, 1200, 1800))
	}))
	defer server.Close()

	covers := NewCoverCache(t.TempDir(), "/covers")
	detail, err := createComic(ctx, db, covers, ComicPayload{
		Series:     "Series",
		Issue:      "1",
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
	if !strings.HasSuffix(detail.Body.CoverImage, ".jpg") {
		t.Fatalf("cover image = %q; want optimized jpg extension", detail.Body.CoverImage)
	}

	coverPath := filepath.Join(covers.dir, strings.TrimPrefix(detail.Body.CoverImage, "/covers/"))
	file, err := os.Open(coverPath)
	if err != nil {
		t.Fatalf("open cached cover: %v", err)
	}
	defer file.Close()
	img, format, err := image.Decode(file)
	if err != nil {
		t.Fatalf("decode cached cover: %v", err)
	}
	if format != "jpeg" {
		t.Fatalf("cached cover format = %q; want jpeg", format)
	}
	if got := max(img.Bounds().Dx(), img.Bounds().Dy()); got != coverMaxDimension {
		t.Fatalf("cached cover max dimension = %d; want %d", got, coverMaxDimension)
	}
}

func TestImportMetronComicDownloadsRemoteCover(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(testJPEG(t, 300, 450))
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

func TestLocalCoverURLSkipsOversizedRemoteCover(t *testing.T) {
	ctx := testUserContext()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		chunk := bytes.Repeat([]byte{0}, 1024)
		for written := 0; written <= coverDownloadMaxBytes; written += len(chunk) {
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	cover, err := localCoverURL(ctx, NewCoverCache(t.TempDir(), "/covers"), server.URL+"/too-large.jpg")
	if err != nil {
		t.Fatalf("localCoverURL: %v", err)
	}
	if cover != "" {
		t.Fatalf("cover = %q; want empty cover for oversized remote image", cover)
	}
}

func TestSyncMetronCharactersDownloadsRemoteImage(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(testPNG(t, 400, 600))
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
		Issue:      "1",
		Publisher:  "Publisher",
		CoverImage: cover,
	}
}

func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, testImage(width, height)); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return out.Bytes()
}

func testJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := jpeg.Encode(&out, testImage(width, height), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return out.Bytes()
}

func testImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 120, A: 255})
		}
	}
	return img
}
