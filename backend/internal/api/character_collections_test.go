package api

import (
	"context"
	"path/filepath"
	"testing"

	databasepkg "github.com/Lofter1/ComicHero/backend/internal/db"
)

func TestCharacterCollectionsArePrivateAndDeduplicateAppearances(t *testing.T) {
	database, err := databasepkg.Open(filepath.Join(t.TempDir(), "collections.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer database.Close()

	if _, err := database.Exec(`
		INSERT INTO users (id, name) VALUES (2, 'Other');
		INSERT INTO characters (id, name) VALUES (10, 'Peter Parker'), (11, 'Miles Morales');
		INSERT INTO comics (id, series, issue, publisher, cover_date)
		VALUES (20, 'Spider-Verse', '1', 'Marvel', '2020-01-01'),
		       (21, 'Spider-Verse', '2', 'Marvel', '2020-02-01');
		INSERT INTO comic_characters (comic_id, character_id)
		VALUES (20, 10), (20, 11), (21, 11);
	`); err != nil {
		t.Fatalf("seed database: %v", err)
	}

	owner := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	other := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	created, err := createCharacterCollection(owner, database, " Spider-Verse ")
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	if created.Body.Name != "Spider-Verse" {
		t.Fatalf("collection name = %q; want trimmed name", created.Body.Name)
	}
	id := created.Body.ID

	for _, characterID := range []int{10, 11, 11} {
		if _, err := addCharacterCollectionMember(owner, database, id, characterID); err != nil {
			t.Fatalf("add character %d: %v", characterID, err)
		}
	}
	detail, err := getCharacterCollection(owner, database, id)
	if err != nil {
		t.Fatalf("get collection: %v", err)
	}
	if detail.Body.CharacterCount != 2 || len(detail.Body.Characters) != 2 {
		t.Fatalf("character counts = %d/%d; want 2/2", detail.Body.CharacterCount, len(detail.Body.Characters))
	}
	if detail.Body.AppearanceCount != 2 || len(detail.Body.Comics) != 2 {
		t.Fatalf("appearance counts = %d/%d; want shared appearance counted once", detail.Body.AppearanceCount, len(detail.Body.Comics))
	}

	otherList, err := listCharacterCollections(other, database, 0)
	if err != nil {
		t.Fatalf("list other user's collections: %v", err)
	}
	if len(otherList.Body) != 0 {
		t.Fatalf("other user saw %d private collections; want 0", len(otherList.Body))
	}
	if _, err := getCharacterCollection(other, database, id); err == nil {
		t.Fatal("other user fetched a private collection")
	}
}

func TestCharacterCollectionDashboardUsesReleaseDateAndReadProgress(t *testing.T) {
	database, err := databasepkg.Open(filepath.Join(t.TempDir(), "collection-dashboard.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer database.Close()

	if _, err := database.Exec(`
		INSERT INTO characters (id, name) VALUES (10, 'Silk'), (11, 'Spider-Gwen');
		INSERT INTO comics (id, series, issue, publisher, cover_date)
		VALUES (20, 'Silk', '2', 'Marvel', '2021-02-01'),
		       (21, 'Spider-Gwen', '1', 'Marvel', '2021-01-01'),
		       (22, 'Spider-Verse', '3', 'Marvel', '2021-03-01');
		INSERT INTO comic_characters (comic_id, character_id)
		VALUES (20, 10), (21, 11), (22, 10), (22, 11);
		INSERT INTO user_comics (comic_id, user_id, read) VALUES (21, 1, 1);
	`); err != nil {
		t.Fatalf("seed database: %v", err)
	}

	ctx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	created, err := createCharacterCollection(ctx, database, "Spider-Verse")
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	for _, characterID := range []int{10, 11} {
		if _, err := addCharacterCollectionMember(ctx, database, created.Body.ID, characterID); err != nil {
			t.Fatalf("add character: %v", err)
		}
	}
	if _, err := setCharacterCollectionStarted(ctx, database, created.Body.ID, true); err != nil {
		t.Fatalf("start collection: %v", err)
	}

	items, err := dashboardCharacterCollections(ctx, database, 1)
	if err != nil {
		t.Fatalf("load dashboard collections: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("dashboard collection count = %d; want 1", len(items))
	}
	if items[0].Type != "characterCollection" || items[0].NextComic == nil || items[0].NextComic.ID != 20 {
		t.Fatalf("dashboard item = %#v; want collection with February issue next", items[0])
	}
	if items[0].Progress < 0.333 || items[0].Progress > 0.334 {
		t.Fatalf("progress = %f; want one of three distinct comics read", items[0].Progress)
	}
}
