package api

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	appdb "github.com/Lofter1/ComicHero/backend/internal/db"
)

func TestMasterDataProvenanceTracksCreatorAndChanger(t *testing.T) {
	database, err := appdb.Open(filepath.Join(t.TempDir(), "provenance.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := ensureDefaultUser(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	created, err := createComic(ctx, database, nil, ComicPayload{Series: "Series", Issue: "1", Publisher: "Publisher"})
	if err != nil {
		var raw Comic
		rawErr := database.Get(&raw, `SELECT *, 0 AS read, 0 AS skipped FROM comics ORDER BY id DESC LIMIT 1`)
		t.Fatalf("%v (raw fetch: %v)", err, rawErr)
	}
	comic := created.Body.Comic
	if comic.CreatedBy == nil || *comic.CreatedBy != 1 || comic.ChangedBy == nil || *comic.ChangedBy != 1 || comic.CreatedAt == "" || comic.ChangedAt == "" {
		t.Fatalf("created provenance = %+v", comic.MasterDataProvenance)
	}

	time.Sleep(1100 * time.Millisecond)
	updated, err := updateComic(ctx, database, nil, comic.ID, ComicPayload{Series: "Series", Issue: "1", Publisher: "Updated"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Body.ChangedAt <= comic.ChangedAt || updated.Body.ChangedBy == nil || *updated.Body.ChangedBy != 1 {
		t.Fatalf("updated provenance = %+v; created = %+v", updated.Body.MasterDataProvenance, comic.MasterDataProvenance)
	}
}
