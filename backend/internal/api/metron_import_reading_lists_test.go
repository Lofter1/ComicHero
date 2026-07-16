package api

import (
	"testing"

	_ "modernc.org/sqlite"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

func TestImportMetronReadingListReusesExistingOrderAndComics(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)
	list := metron.ReadingList{
		ID:          501,
		Name:        "Event",
		Description: "Big event",
		Image:       "https://example.test/event.jpg",
		Issues: []metron.Issue{
			{
				ID:         101,
				Series:     "Series",
				SeriesYear: 2026,
				Issue:      "1",
				Publisher:  "Publisher",
				Tags:       []string{"Main Story", "Tie-In"},
			},
		},
	}

	first, err := importMetronReadingList(ctx, db, nil, nil, list)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	second, err := importMetronReadingList(ctx, db, nil, nil, list)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if first.Body.ID != second.Body.ID {
		t.Fatalf("order ids differ: first=%d second=%d", first.Body.ID, second.Body.ID)
	}
	if second.Body.Image != "https://example.test/event.jpg" {
		t.Fatalf("image = %q; want Metron image", second.Body.Image)
	}
	if second.Body.Rating != 0 {
		t.Fatalf("rating = %v; want no imported Metron rating", second.Body.Rating)
	}
	if second.Body.RatingCount != 0 {
		t.Fatalf("rating count = %d; want no imported Metron ratings", second.Body.RatingCount)
	}

	var orderCount int
	if err := db.GetContext(ctx, &orderCount, `SELECT COUNT(*) FROM reading_orders`); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if orderCount != 1 {
		t.Fatalf("order count = %d; want 1", orderCount)
	}

	var comicCount int
	if err := db.GetContext(ctx, &comicCount, `SELECT COUNT(*) FROM comics`); err != nil {
		t.Fatalf("count comics: %v", err)
	}
	if comicCount != 1 {
		t.Fatalf("comic count = %d; want 1", comicCount)
	}

	var tags string
	if err := db.GetContext(ctx, &tags, `SELECT tags FROM reading_order_comics WHERE reading_order_id = ?`, first.Body.ID); err != nil {
		t.Fatalf("select tags: %v", err)
	}
	if tags != "Main Story, Tie-In" {
		t.Fatalf("tags = %q; want Main Story, Tie-In", tags)
	}
}

func TestContinueMetronReadingListFillsExistingOrder(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO reading_orders (id, name, description, metron_reading_list_id)
		VALUES (1, 'Event', 'Partial import', 501)
	`); err != nil {
		t.Fatalf("insert incomplete reading order: %v", err)
	}

	err := continueMetronReadingListWithProgress(ctx, db, nil, nil, metron.ReadingList{
		ID:          501,
		Name:        "Event",
		Description: "Big event",
		Image:       "https://example.test/event.jpg",
		Issues: []metron.Issue{
			{
				ID:         101,
				Series:     "Series",
				SeriesYear: 2026,
				Issue:      "1",
				Publisher:  "Publisher",
			},
		},
	}, func(int, int, string) {})
	if err != nil {
		t.Fatalf("continueMetronReadingListWithProgress: %v", err)
	}

	var comicCount int
	if err := db.GetContext(ctx, &comicCount, `SELECT COUNT(*) FROM comics`); err != nil {
		t.Fatalf("count comics: %v", err)
	}
	if comicCount != 1 {
		t.Fatalf("comic count = %d; want 1", comicCount)
	}
	var image string
	if err := db.GetContext(ctx, &image, `SELECT image FROM reading_orders WHERE id = 1`); err != nil {
		t.Fatalf("select image: %v", err)
	}
	if image != "https://example.test/event.jpg" {
		t.Fatalf("image = %q; want Metron image", image)
	}

	var linkedCount int
	if err := db.GetContext(ctx, &linkedCount, `SELECT COUNT(*) FROM reading_order_comics WHERE reading_order_id = 1`); err != nil {
		t.Fatalf("count linked comics: %v", err)
	}
	if linkedCount != 1 {
		t.Fatalf("linked comics = %d; want 1", linkedCount)
	}
}

func TestMetronReadingListLinksComicsDuringImportProgress(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	progressLinkedCounts := []int{}
	err := importMetronReadingListWithOptions(ctx, db, nil, nil, metron.ReadingList{
		ID:          501,
		Name:        "Event",
		Description: "Big event",
		Issues: []metron.Issue{
			{
				ID:         101,
				Series:     "Series",
				SeriesYear: 2026,
				Issue:      "1",
				Publisher:  "Publisher",
			},
			{
				ID:         102,
				Series:     "Series",
				SeriesYear: 2026,
				Issue:      "2",
				Publisher:  "Publisher",
			},
		},
	}, false, func(completed, total int, message string) {
		if completed <= 0 {
			return
		}
		var linkedCount int
		if err := db.GetContext(ctx, &linkedCount, `
			SELECT COUNT(*) FROM reading_order_comics roc
			JOIN reading_orders ro ON ro.id = roc.reading_order_id
			WHERE ro.metron_reading_list_id = 501
		`); err != nil {
			t.Fatalf("count linked comics during progress: %v", err)
		}
		progressLinkedCounts = append(progressLinkedCounts, linkedCount)
	}, defaultMetronImportOptions())
	if err != nil {
		t.Fatalf("importMetronReadingListWithOptions: %v", err)
	}

	if len(progressLinkedCounts) < 2 {
		t.Fatalf("progress linked counts = %#v; want counts during each issue", progressLinkedCounts)
	}
	if progressLinkedCounts[0] != 1 || progressLinkedCounts[1] != 2 {
		t.Fatalf("progress linked counts = %#v; want [1 2]", progressLinkedCounts)
	}
}

func TestUpdateComicReadStatus(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO comics (id, series, series_year, issue, publisher, read)
		VALUES (1, 'Series', 2026, 1, 'Publisher', 0)
	`); err != nil {
		t.Fatalf("insert comic: %v", err)
	}

	read := true
	skipped := true
	input := &UpdateComicReadInput{ID: 1}
	input.Body.Read = &read
	input.Body.Skipped = &skipped
	detail, err := updateComicReadStatus(ctx, db, 1, input)
	if err != nil {
		t.Fatalf("updateComicReadStatus: %v", err)
	}
	if !detail.Body.Read {
		t.Fatal("comic read status was not updated")
	}
	if !detail.Body.Skipped {
		t.Fatal("comic skipped status was not updated")
	}
	if detail.Body.Title != "Series (2026) #1" {
		t.Fatalf("comic metadata changed unexpectedly: %#v", detail.Body)
	}

	var stored struct {
		Read    int `db:"read"`
		Skipped int `db:"skipped"`
	}
	if err := db.GetContext(ctx, &stored, `
		SELECT read, skipped FROM user_comics WHERE comic_id = ? AND user_id = (
			SELECT id FROM users WHERE name = 'Default'
		)
	`, 1); err != nil {
		t.Fatalf("read status row lookup: %v", err)
	}
	if stored.Read != 1 {
		t.Fatalf("stored read flag = %d; want 1", stored.Read)
	}
	if stored.Skipped != 1 {
		t.Fatalf("stored skipped flag = %d; want 1", stored.Skipped)
	}
}
