package api

import (
	"context"
	"encoding/base64"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func TestReadingOrderHelpers(t *testing.T) {
	progress := computeProgress([]ReadingOrderComic{
		{Comic: Comic{Read: true}},
		{Comic: Comic{Read: false}},
	})
	if progress != 0.5 {
		t.Fatalf("progress = %v; want 0.5", progress)
	}

	input := &SetReadingOrderComicsInput{}
	input.Body.ComicIDs = []int{1, 2}

	items := readingOrderComicItems(input)
	if len(items) != 2 || items[0].ComicID != 1 || items[1].ComicID != 2 {
		t.Fatalf("items = %#v; want comic IDs 1, 2", items)
	}

	ids := readingOrderComicIDs([]ReadingOrderComicPayload{
		{ComicID: 3},
		{ComicID: 3},
	})
	if len(ids) != 2 || ids[0] != 3 || ids[1] != 3 {
		t.Fatalf("ids = %#v; want duplicate IDs preserved", ids)
	}

	if order := readingOrderListOrder("rating", "desc"); order != "ORDER BY rating DESC, ro.name DESC" {
		t.Fatalf("rating order = %q", order)
	}

	section := normalizeReadingOrderEntry(ReadingOrderEntryPayload{
		Type:  "section",
		Title: "  Main story  ",
	})
	if section.Title != "Main story" {
		t.Fatalf("normalized section title = %q; want Main story", section.Title)
	}
	if err := validateReadingOrderEntries([]ReadingOrderEntryPayload{section}); err != nil {
		t.Fatalf("validate section: %v", err)
	}
	if err := validateReadingOrderEntries([]ReadingOrderEntryPayload{{Type: "section"}}); err == nil {
		t.Fatal("validate untitled section = nil; want an error")
	}
}

func TestReadingOrderEntriesCanNestOrdersBetweenComics(t *testing.T) {
	ctx := testUserContext()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE reading_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			is_public INTEGER NOT NULL DEFAULT 1,
			metron_reading_list_id INTEGER,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0,
			rating REAL NOT NULL DEFAULT 0,
			rating_count INTEGER NOT NULL DEFAULT 0,
			author_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL
		);
		CREATE TABLE user_reading_orders (
			reading_order_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT,
			favorite INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (reading_order_id, user_id)
		);
		CREATE TABLE reading_order_ratings (
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			rating REAL NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (reading_order_id, user_id)
		);
		CREATE TABLE user_arcs (arc_id INTEGER NOT NULL, user_id INTEGER NOT NULL, started_at TEXT, favorite INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (arc_id, user_id));
		CREATE TABLE comics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			series_id INTEGER REFERENCES series(id) ON DELETE SET NULL,
			series TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0,
			issue TEXT NOT NULL,
			publisher TEXT NOT NULL,
			cover_date TEXT NOT NULL DEFAULT '',
			cover_image TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			read INTEGER NOT NULL DEFAULT 0,
			metron_issue_id INTEGER
		);
			CREATE TABLE users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL UNIQUE,
					email TEXT NOT NULL DEFAULT '',
					email_verified_at TEXT NOT NULL DEFAULT '2026-01-01T00:00:00Z',
					password_hash TEXT NOT NULL DEFAULT '',
				is_default INTEGER NOT NULL DEFAULT 0,
				is_admin INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE user_comics (
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			read INTEGER NOT NULL DEFAULT 0,
			skipped INTEGER NOT NULL DEFAULT 0,
			read_at TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (comic_id, user_id)
		);
		INSERT OR IGNORE INTO users (id, name, is_default) VALUES (1, 'Default', 1);
		CREATE TABLE reading_order_comics (
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE reading_order_children (
			parent_reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			child_reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (parent_reading_order_id, child_reading_order_id),
			CHECK (parent_reading_order_id <> child_reading_order_id)
		);
		CREATE TABLE reading_order_sections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT ''
		);
		INSERT INTO comics (series, series_year, issue, publisher)
		VALUES ('Parent', 2026, '1', 'Publisher'),
			('Child', 2026, '2', 'Publisher'),
			('Parent', 2026, '3', 'Publisher');
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	parent, err := createReadingOrder(ctx, db, nil, ReadingOrderPayload{Name: "Parent"})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	child, err := createReadingOrder(ctx, db, nil, ReadingOrderPayload{Name: "Child"})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	childInput := &SetReadingOrderComicsInput{ID: child.Body.ID}
	childInput.Body.Entries = []ReadingOrderEntryPayload{{Type: "comic", ComicID: 2}}
	if _, err := setReadingOrderComics(ctx, db, childInput); err != nil {
		t.Fatalf("set child entries: %v", err)
	}

	parentInput := &SetReadingOrderComicsInput{ID: parent.Body.ID}
	parentInput.Body.Entries = []ReadingOrderEntryPayload{
		{Type: "section", Title: "Opening", Description: "Start with the setup."},
		{Type: "comic", ComicID: 1},
		{Type: "readingOrder", ReadingOrderID: child.Body.ID, Comment: "Crossover break"},
		{Type: "section", Title: "Finale"},
		{Type: "comic", ComicID: 3},
	}
	detail, err := setReadingOrderComics(ctx, db, parentInput)
	if err != nil {
		t.Fatalf("set parent entries: %v", err)
	}

	if len(detail.Body.Entries) != 5 || detail.Body.Entries[0].Type != "section" || detail.Body.Entries[2].Type != "readingOrder" {
		t.Fatalf("entries = %#v; want sections around ordered comic entries", detail.Body.Entries)
	}
	if detail.Body.Entries[0].Section == nil || detail.Body.Entries[0].Section.Title != "Opening" || detail.Body.Entries[0].Section.Description != "Start with the setup." {
		t.Fatalf("opening section = %#v", detail.Body.Entries[0].Section)
	}
	if detail.Body.Entries[2].Comment != "Crossover break" {
		t.Fatalf("nested order note = %q; want Crossover break", detail.Body.Entries[2].Comment)
	}
	if len(detail.Body.Entries[2].Comics) != 1 || detail.Body.Entries[2].Comics[0].Issue != "2" {
		t.Fatalf("nested expanded comics = %#v; want issue 2", detail.Body.Entries[2].Comics)
	}
	if len(detail.Body.Comics) != 3 {
		t.Fatalf("expanded comics = %d; want 3", len(detail.Body.Comics))
	}
	for i, issue := range []string{"1", "2", "3"} {
		if detail.Body.Comics[i].Issue != issue {
			t.Fatalf("comic %d issue = %q; want %q", i, detail.Body.Comics[i].Issue, issue)
		}
	}
	if detail.Body.Comics[1].Comment != "From Child: Crossover break" {
		t.Fatalf("nested comic comment = %q; want nested order note", detail.Body.Comics[1].Comment)
	}
}

func TestCopyReadingOrderCreatesCurrentUserOwnedCopy(t *testing.T) {
	db := setupReadingOrderCBLTestDB(t)
	ctx := testUserContext()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO users (id, name, is_default) VALUES (2, 'Other', 0);
		INSERT INTO comics (id, series, series_year, issue, publisher)
		VALUES
			(1, 'Source', 2026, '1', 'Publisher'),
			(2, 'Source', 2026, '2', 'Publisher');
		INSERT INTO reading_orders (id, name, description, image, favorite, author_user_id)
		VALUES
			(10, 'Other list', 'Read this one', '/covers/list.jpg', 1, 2),
			(11, 'Nested list', 'Nested notes', '', 0, 2);
		INSERT INTO reading_order_comics (reading_order_id, comic_id, position, note, tags)
		VALUES
			(10, 1, 2, 'Start here', 'Main'),
			(11, 2, 1, 'Nested comic', 'Tie-in');
		INSERT INTO reading_order_children (parent_reading_order_id, child_reading_order_id, position, note)
		VALUES (10, 11, 3, 'Then this');
		INSERT INTO reading_order_sections (reading_order_id, position, title, description)
		VALUES (10, 1, 'First phase', 'Read the setup first');
	`); err != nil {
		t.Fatalf("seed copy source: %v", err)
	}

	copied, err := copyReadingOrder(ctx, db, 10)
	if err != nil {
		t.Fatalf("copyReadingOrder: %v", err)
	}

	if copied.Body.ID == 10 {
		t.Fatal("copied reading order reused source id")
	}
	if copied.Body.AuthorUserID == nil || *copied.Body.AuthorUserID != 1 {
		t.Fatalf("copied author = %#v; want current user 1", copied.Body.AuthorUserID)
	}
	if copied.Body.Name != "Other list (Copy)" {
		t.Fatalf("copied name = %q; want copy suffix", copied.Body.Name)
	}
	if copied.Body.Description != "Read this one" || copied.Body.Image != "/covers/list.jpg" {
		t.Fatalf("copied metadata = description %q image %q", copied.Body.Description, copied.Body.Image)
	}
	if copied.Body.Favorite {
		t.Fatal("copied favorite = true; want false for current user's new copy")
	}
	if !copied.Body.CanEdit {
		t.Fatal("copied canEdit = false; want current user to edit their copy")
	}
	if len(copied.Body.Entries) != 3 {
		t.Fatalf("copied entries = %d; want 3", len(copied.Body.Entries))
	}
	if copied.Body.Entries[0].Section == nil || copied.Body.Entries[0].Section.Title != "First phase" || copied.Body.Entries[0].Section.Description != "Read the setup first" {
		t.Fatalf("first copied entry = %#v; want section", copied.Body.Entries[0])
	}
	if copied.Body.Entries[1].Comic == nil || copied.Body.Entries[1].Comic.ID != 1 {
		t.Fatalf("second copied entry = %#v; want comic 1", copied.Body.Entries[1])
	}
	if copied.Body.Entries[1].Comic.Comment != "Start here" || copied.Body.Entries[1].Comic.Tags != "Main" {
		t.Fatalf("copied comic note/tags = %q/%q", copied.Body.Entries[1].Comic.Comment, copied.Body.Entries[1].Comic.Tags)
	}
	if copied.Body.Entries[2].ReadingOrder == nil || copied.Body.Entries[2].ReadingOrder.ID != 11 {
		t.Fatalf("third copied entry = %#v; want nested order 11", copied.Body.Entries[2])
	}
	if copied.Body.Entries[2].Comment != "Then this" {
		t.Fatalf("copied nested note = %q; want Then this", copied.Body.Entries[2].Comment)
	}
}

func TestReadingOrderWritesRequireAuthor(t *testing.T) {
	db := setupReadingOrderCBLTestDB(t)
	ctx := testUserContext()
	ownerCtx := context.WithValue(ctx, contextUserIDKey{}, 2)
	otherCtx := context.WithValue(ctx, contextUserIDKey{}, 1)
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, name, is_default) VALUES (2, 'Owner', 0)`); err != nil {
		t.Fatalf("insert owner: %v", err)
	}

	created, err := createReadingOrder(ownerCtx, db, nil, ReadingOrderPayload{Name: "Owner list"})
	if err != nil {
		t.Fatalf("createReadingOrder: %v", err)
	}
	if created.Body.AuthorUserID == nil || *created.Body.AuthorUserID != 2 {
		t.Fatalf("author user id = %#v; want 2", created.Body.AuthorUserID)
	}

	view, err := getReadingOrder(otherCtx, db, created.Body.ID)
	if err != nil {
		t.Fatalf("getReadingOrder as non-author: %v", err)
	}
	if view.Body.CanEdit {
		t.Fatalf("non-author canEdit = true; want false")
	}

	if _, err := updateReadingOrder(otherCtx, db, nil, created.Body.ID, ReadingOrderPayload{Name: "Nope"}); err == nil {
		t.Fatalf("updateReadingOrder as non-author succeeded; want error")
	}
	if _, err := db.ExecContext(ctx, `UPDATE users SET is_admin = 1 WHERE id = 1`); err != nil {
		t.Fatalf("promote admin: %v", err)
	}
	adminView, err := getReadingOrder(otherCtx, db, created.Body.ID)
	if err != nil {
		t.Fatalf("getReadingOrder as admin: %v", err)
	}
	if !adminView.Body.CanEdit {
		t.Fatalf("admin canEdit = false; want true")
	}
	if _, err := updateReadingOrder(otherCtx, db, nil, created.Body.ID, ReadingOrderPayload{Name: "Admin update"}); err != nil {
		t.Fatalf("updateReadingOrder as admin: %v", err)
	}
	if _, err := updateReadingOrder(ownerCtx, db, nil, created.Body.ID, ReadingOrderPayload{Name: "Updated"}); err != nil {
		t.Fatalf("updateReadingOrder as author: %v", err)
	}
}

func TestReadingOrderEngagementCountsAndSort(t *testing.T) {
	db := setupReadingOrderCBLTestDB(t)
	ctx := testUserContext()
	for _, user := range []struct {
		id   int
		name string
	}{{2, "Reader Two"}, {3, "Reader Three"}} {
		if _, err := db.Exec(`INSERT INTO users (id, name) VALUES (?, ?)`, user.id, user.name); err != nil {
			t.Fatalf("insert user %d: %v", user.id, err)
		}
	}

	popular, err := createReadingOrder(ctx, db, nil, ReadingOrderPayload{Name: "Popular"})
	if err != nil {
		t.Fatalf("create popular reading order: %v", err)
	}
	quiet, err := createReadingOrder(ctx, db, nil, ReadingOrderPayload{Name: "Quiet"})
	if err != nil {
		t.Fatalf("create quiet reading order: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO user_reading_orders (reading_order_id, user_id, favorite, started_at)
		VALUES (?, 2, 1, CURRENT_TIMESTAMP), (?, 3, 1, NULL), (?, 2, 1, NULL)
	`, popular.Body.ID, popular.Body.ID, quiet.Body.ID); err != nil {
		t.Fatalf("insert reading order preferences: %v", err)
	}

	result, err := listReadingOrders(ctx, db, &ReadingOrderListInput{
		Sort:      "favoriteCount",
		Direction: "desc",
	})
	if err != nil {
		t.Fatalf("list reading orders by favorites: %v", err)
	}
	if len(result.Body) != 2 || result.Body[0].ID != popular.Body.ID {
		t.Fatalf("favorite sort result = %#v; want popular reading order first", result.Body)
	}
	if result.Body[0].FavoriteCount != 2 || result.Body[0].StartedCount != 1 {
		t.Fatalf("popular counts = %d favorites, %d started; want 2/1", result.Body[0].FavoriteCount, result.Body[0].StartedCount)
	}
}

func TestEngagementMetricSortOrders(t *testing.T) {
	for _, test := range []struct {
		name string
		got  string
		want string
	}{
		{"reading order favorites", readingOrderListOrder("favoriteCount", "desc"), "favorite_count"},
		{"reading order started", readingOrderListOrder("startedCount", "desc"), "started_count"},
		{"arc favorites", arcListOrder("favoriteCount", "desc"), "favorite_count"},
		{"arc started", arcListOrder("startedCount", "desc"), "started_count"},
		{"character favorites", characterListOrder("favoriteCount", "desc"), "favorite_count"},
		{"character started", characterListOrder("startedCount", "desc"), "started_count"},
		{"series favorites", seriesListOrder("favoriteCount", "desc"), "favorite_count"},
		{"series started", seriesListOrder("startedCount", "desc"), "started_count"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !strings.Contains(test.got, test.want+" DESC") {
				t.Fatalf("sort order = %q; want descending %s", test.got, test.want)
			}
		})
	}
}

func TestPrivateReadingOrderIsOnlyVisibleToCreatorAndAdmin(t *testing.T) {
	db := setupReadingOrderCBLTestDB(t)
	ctx := testUserContext()
	ownerCtx := context.WithValue(ctx, contextUserIDKey{}, 2)
	otherCtx := context.WithValue(ctx, contextUserIDKey{}, 1)
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, name, is_default) VALUES (2, 'Owner', 0)`); err != nil {
		t.Fatalf("insert owner: %v", err)
	}

	isPublic := false
	created, err := createReadingOrder(ownerCtx, db, nil, ReadingOrderPayload{
		Name:     "Private list",
		IsPublic: &isPublic,
	})
	if err != nil {
		t.Fatalf("create private reading order: %v", err)
	}
	if created.Body.IsPublic {
		t.Fatal("created reading order is public; want private")
	}
	if _, err := getReadingOrder(ownerCtx, db, created.Body.ID); err != nil {
		t.Fatalf("creator cannot view private reading order: %v", err)
	}
	if _, err := getReadingOrder(otherCtx, db, created.Body.ID); err == nil {
		t.Fatal("non-creator can view private reading order")
	}

	listed, err := listReadingOrders(otherCtx, db, &ReadingOrderListInput{})
	if err != nil {
		t.Fatalf("list reading orders as non-creator: %v", err)
	}
	if len(listed.Body) != 0 {
		t.Fatalf("non-creator list = %#v; want private reading order hidden", listed.Body)
	}

	if _, err := db.ExecContext(ctx, `UPDATE users SET is_admin = 1 WHERE id = 1`); err != nil {
		t.Fatalf("promote admin: %v", err)
	}
	if _, err := getReadingOrder(otherCtx, db, created.Body.ID); err != nil {
		t.Fatalf("admin cannot view private reading order: %v", err)
	}
}

func TestReadingOrderCoverUploadResizesAndDeletesUnusedOldCover(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)
	covers := NewCoverCache(t.TempDir(), "/covers")

	created, err := createReadingOrder(ctx, db, covers, ReadingOrderPayload{
		Name:           "Cover list",
		CoverImageData: imageDataURL("image/png", testPNG(t, 1200, 1800)),
	})
	if err != nil {
		t.Fatalf("createReadingOrder: %v", err)
	}
	oldImage := created.Body.Image
	if oldImage == "" {
		t.Fatal("created reading order missing cover image")
	}
	oldPath := coverPath(t, covers, oldImage)
	assertCoverMaxDimension(t, oldPath, coverMaxDimension)

	updated, err := updateReadingOrder(ctx, db, covers, created.Body.ID, ReadingOrderPayload{
		Name:           "Cover list",
		CoverImageData: imageDataURL("image/jpeg", testJPEG(t, 300, 450)),
	})
	if err != nil {
		t.Fatalf("updateReadingOrder: %v", err)
	}
	if updated.Body.Image == "" {
		t.Fatal("updated reading order missing cover image")
	}
	if updated.Body.Image == oldImage {
		t.Fatalf("image was not replaced: %q", updated.Body.Image)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old cover still exists or stat failed: %v", err)
	}
	assertCoverMaxDimension(t, coverPath(t, covers, updated.Body.Image), 450)
}

func TestReadingOrderCBLImportPrefersComicVineThenFallsBackAndCreatesMissingComics(t *testing.T) {
	ctx := testUserContext()
	db := setupReadingOrderCBLTestDB(t)

	if _, err := db.Exec(`
		INSERT INTO comics (series, series_year, issue, publisher, cover_date, comic_vine_id)
		VALUES ('Frank Miller''s RoboCop', 2003, '1', 'Avatar Press', '2003-07-01', 111),
			('Frank Miller''s RoboCop', 2003, '2', 'Avatar Press', '2003-08-01', NULL);
	`); err != nil {
		t.Fatalf("seed comics: %v", err)
	}

	input := &ReadingOrderCBLImportInput{}
	input.Body.Filename = "fallback.cbl"
	input.Body.Content = `<?xml version="1.0" encoding="utf-8"?>
<ReadingList>
	<Name>RoboCop Publication Order</Name>
	<NumIssues>5</NumIssues>
	<Books>
		<Book Series="Wrong metadata" Number="99" Volume="1999"><Database Name="cv" Issue="4000-111" /></Book>
		<Book Series="Frank Miller&apos;s RoboCop" Number="2" Volume="2003" Year="2003"><Database Name="Comic Vine" Issue="222" /></Book>
		<Book Series="Missing Series" Number="1" Volume="2003" Year="2003"><Database Name="comicvine" Issue="333" /></Book>
		<Book Series="No CV Series" Number="5" Volume="2024" Year="2024" />
		<Book Number="4" Volume="2003" Year="2003" />
	</Books>
	<Matchers />
</ReadingList>`

	result, err := importReadingOrderCBL(ctx, db, input)
	if err != nil {
		t.Fatalf("importReadingOrderCBL: %v", err)
	}

	if result.Body.ReadingOrder.Name != "RoboCop Publication Order" {
		t.Fatalf("name = %q; want CBL name", result.Body.ReadingOrder.Name)
	}
	if result.Body.MatchedCount != 4 || result.Body.UnmatchedCount != 1 {
		t.Fatalf("matched/unmatched = %d/%d; want 4/1", result.Body.MatchedCount, result.Body.UnmatchedCount)
	}
	if len(result.Body.ReadingOrder.Comics) != 4 {
		t.Fatalf("imported comics = %d; want 4", len(result.Body.ReadingOrder.Comics))
	}
	if result.Body.ReadingOrder.Comics[0].ID != 1 || result.Body.ReadingOrder.Comics[1].ID != 2 || result.Body.ReadingOrder.Comics[2].ID != 3 || result.Body.ReadingOrder.Comics[3].ID != 4 {
		t.Fatalf("imported comic IDs = %#v; want existing 1, existing 2, then new 3 and 4", result.Body.ReadingOrder.Comics)
	}
	if len(result.Body.Unmatched) != 1 || result.Body.Unmatched[0].Reason != "missing series or issue number" {
		t.Fatalf("unmatched = %#v; want malformed book", result.Body.Unmatched)
	}
	var comicVineIDs []int
	if err := db.Select(&comicVineIDs, `SELECT COALESCE(comic_vine_id, 0) FROM comics ORDER BY id`); err != nil {
		t.Fatalf("read Comic Vine IDs: %v", err)
	}
	if len(comicVineIDs) != 4 || comicVineIDs[0] != 111 || comicVineIDs[1] != 222 || comicVineIDs[2] != 333 || comicVineIDs[3] != 0 {
		t.Fatalf("Comic Vine IDs = %#v; want 111, 222, 333, 0", comicVineIDs)
	}
}

func TestReadingOrderCBLImportGroupsMultipartFilesInPartOrder(t *testing.T) {
	ctx := testUserContext()
	db := setupReadingOrderCBLTestDB(t)

	input := &ReadingOrderCBLImportInput{}
	input.Body.Parts = []ReadingOrderCBLImportPart{
		{
			Filename: "[Marvel] CMRO Core Reading Order-Part 02.cbl",
			Content: `<ReadingList>
	<Name>[Marvel] CMRO Core Reading Order-Part 02</Name>
	<Books>
		<Book Series="Fantastic Four" Number="2" Volume="1961" Year="1962"><Database Name="cv" Issue="5712" /></Book>
		<Book Number="3" Volume="1961" Year="1962" />
	</Books>
</ReadingList>`,
		},
		{
			Filename: "[Marvel] CMRO Core Reading Order-Part 01.cbl",
			Content: `<ReadingList>
	<Name>[Marvel] CMRO Core Reading Order-Part 01</Name>
	<Books>
		<Book Series="Fantastic Four" Number="1" Volume="1961" Year="1961"><Database Name="cv" Issue="5558" /></Book>
	</Books>
</ReadingList>`,
		},
	}

	result, err := importReadingOrderCBL(ctx, db, input)
	if err != nil {
		t.Fatalf("importReadingOrderCBL: %v", err)
	}

	if result.Body.ReadingOrder.Name != "[Marvel] CMRO Core Reading Order" {
		t.Fatalf("parent name = %q; want suffix-free multipart name", result.Body.ReadingOrder.Name)
	}
	if result.Body.MatchedCount != 2 || result.Body.UnmatchedCount != 1 {
		t.Fatalf("matched/unmatched = %d/%d; want 2/1", result.Body.MatchedCount, result.Body.UnmatchedCount)
	}
	if len(result.Body.Unmatched) != 1 || result.Body.Unmatched[0].Part != "[Marvel] CMRO Core Reading Order-Part 02" {
		t.Fatalf("unmatched = %#v; want source part on malformed book", result.Body.Unmatched)
	}

	order := result.Body.ReadingOrder
	if len(order.ChildReadingOrders) != 0 || len(order.Entries) != 4 || len(order.Comics) != 2 {
		t.Fatalf("multipart children/entries/comics = %d/%d/%d; want 0/4/2", len(order.ChildReadingOrders), len(order.Entries), len(order.Comics))
	}
	wantPartNames := []string{
		"[Marvel] CMRO Core Reading Order-Part 01",
		"[Marvel] CMRO Core Reading Order-Part 02",
	}
	for i, want := range wantPartNames {
		entry := order.Entries[i*2]
		if entry.Type != "section" || entry.Section == nil || entry.Section.Title != want {
			t.Fatalf("part section %d = %#v; want %q", i, entry, want)
		}
	}
	if order.Comics[0].Issue != "1" || order.Comics[1].Issue != "2" {
		t.Fatalf("flattened issues = %q, %q; want part order 1, 2", order.Comics[0].Issue, order.Comics[1].Issue)
	}
	var readingOrderCount int
	if err := db.Get(&readingOrderCount, `SELECT COUNT(*) FROM reading_orders`); err != nil {
		t.Fatalf("count reading orders: %v", err)
	}
	if readingOrderCount != 1 {
		t.Fatalf("reading order count = %d; want one combined multipart order", readingOrderCount)
	}
}

func TestMultipartCBLImportRejectsUnrelatedReadingLists(t *testing.T) {
	ctx := testUserContext()
	db := setupReadingOrderCBLTestDB(t)
	input := &ReadingOrderCBLImportInput{}
	input.Body.Parts = []ReadingOrderCBLImportPart{
		{Filename: "First-Part 01.cbl", Content: `<ReadingList><Name>First-Part 01</Name></ReadingList>`},
		{Filename: "Second-Part 02.cbl", Content: `<ReadingList><Name>Second-Part 02</Name></ReadingList>`},
	}

	if _, err := importReadingOrderCBL(ctx, db, input); err == nil {
		t.Fatal("importReadingOrderCBL succeeded; want unrelated multipart names rejected")
	}
	var readingOrderCount int
	if err := db.Get(&readingOrderCount, `SELECT COUNT(*) FROM reading_orders`); err != nil {
		t.Fatalf("count reading orders: %v", err)
	}
	if readingOrderCount != 0 {
		t.Fatalf("reading order count = %d; want validation before creating data", readingOrderCount)
	}
}

func imageDataURL(mime string, data []byte) string {
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func coverPath(t *testing.T, covers *CoverCache, image string) string {
	t.Helper()
	name, ok := covers.localFileName(image)
	if !ok {
		t.Fatalf("cover URL %q is not local", image)
	}
	return filepath.Join(covers.dir, name)
}

func assertCoverMaxDimension(t *testing.T, path string, want int) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open cover: %v", err)
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		t.Fatalf("decode cover: %v", err)
	}
	if got := max(img.Bounds().Dx(), img.Bounds().Dy()); got != want {
		t.Fatalf("cover max dimension = %d; want %d", got, want)
	}
}

func TestReadingOrderCBLExportBuildsFlatCBLXML(t *testing.T) {
	ctx := testUserContext()
	db := setupReadingOrderCBLTestDB(t)

	metronID := 98765
	comicVineID := 54321
	if _, err := db.Exec(`
		INSERT INTO comics (series, series_year, issue, publisher, cover_date, metron_issue_id, comic_vine_id)
		VALUES ('Batman & Robin', 2011, '1', 'DC Comics', '2011-09-01', ?, ?),
			('Batman & Robin', 2011, '2', 'DC Comics', '2011-10-01', NULL, NULL);
	`, metronID, comicVineID); err != nil {
		t.Fatalf("seed comics: %v", err)
	}

	created, err := createReadingOrder(ctx, db, nil, ReadingOrderPayload{Name: "Batman & Robin"})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	input := &SetReadingOrderComicsInput{ID: created.Body.ID}
	input.Body.Entries = []ReadingOrderEntryPayload{
		{Type: "comic", ComicID: 1},
		{Type: "comic", ComicID: 2},
	}
	if _, err := setReadingOrderComics(ctx, db, input); err != nil {
		t.Fatalf("set entries: %v", err)
	}

	result, err := exportReadingOrderCBL(ctx, db, created.Body.ID)
	if err != nil {
		t.Fatalf("exportReadingOrderCBL: %v", err)
	}

	for _, fragment := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<Name>Batman &amp; Robin</Name>`,
		`<NumIssues>2</NumIssues>`,
		`Series="Batman &amp; Robin" Number="1" Volume="2011" Year="2011"`,
		`<Database Name="comicvine" Issue="54321"></Database>`,
		`<Database Name="metron" Issue="98765"></Database>`,
	} {
		if !strings.Contains(result.Body.Content, fragment) {
			t.Fatalf("export missing %q in:\n%s", fragment, result.Body.Content)
		}
	}
	if result.Body.Filename != "Batman-Robin.cbl" {
		t.Fatalf("filename = %q; want Batman-Robin.cbl", result.Body.Filename)
	}
}

func setupReadingOrderCBLTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE reading_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			is_public INTEGER NOT NULL DEFAULT 1,
			metron_reading_list_id INTEGER,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0,
			rating REAL NOT NULL DEFAULT 0,
			rating_count INTEGER NOT NULL DEFAULT 0,
			author_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL
		);
		CREATE TABLE user_reading_orders (
			reading_order_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT,
			favorite INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (reading_order_id, user_id)
		);
		CREATE TABLE reading_order_ratings (
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			rating REAL NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (reading_order_id, user_id)
		);
		CREATE TABLE series (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0
		);
		CREATE UNIQUE INDEX idx_series_name_year ON series(name, series_year);
		CREATE TABLE comics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			series_id INTEGER REFERENCES series(id) ON DELETE SET NULL,
			series TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0,
			issue TEXT NOT NULL,
			publisher TEXT NOT NULL,
			cover_date TEXT NOT NULL DEFAULT '',
			cover_image TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			read INTEGER NOT NULL DEFAULT 0,
			metron_issue_id INTEGER,
			comic_vine_id INTEGER
		);
		CREATE UNIQUE INDEX idx_comics_comic_vine_id
		ON comics(comic_vine_id)
		WHERE comic_vine_id IS NOT NULL;
			CREATE TABLE users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL UNIQUE,
					email TEXT NOT NULL DEFAULT '',
					email_verified_at TEXT NOT NULL DEFAULT '2026-01-01T00:00:00Z',
					password_hash TEXT NOT NULL DEFAULT '',
				is_default INTEGER NOT NULL DEFAULT 0,
				is_admin INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE user_comics (
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			read INTEGER NOT NULL DEFAULT 0,
			skipped INTEGER NOT NULL DEFAULT 0,
			read_at TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (comic_id, user_id)
		);
		INSERT OR IGNORE INTO users (id, name, is_default) VALUES (1, 'Default', 1);
		CREATE TABLE reading_order_comics (
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE reading_order_children (
			parent_reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			child_reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (parent_reading_order_id, child_reading_order_id),
			CHECK (parent_reading_order_id <> child_reading_order_id)
		);
		CREATE TABLE reading_order_sections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT ''
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func TestRateReadingOrderUsesUserRatings(t *testing.T) {
	ctx := testUserContext()
	db := setupReadingOrderCBLTestDB(t)
	if _, err := db.Exec(`INSERT INTO users (id, name) VALUES (2, 'Second Reader')`); err != nil {
		t.Fatalf("insert second user: %v", err)
	}

	created, err := createReadingOrder(ctx, db, nil, ReadingOrderPayload{Name: "Rated order"})
	if err != nil {
		t.Fatalf("create reading order: %v", err)
	}

	rated, err := rateReadingOrder(ctx, db, created.Body.ID, 4)
	if err != nil {
		t.Fatalf("rate reading order: %v", err)
	}
	if rated.Body.Rating != 4 || rated.Body.RatingCount != 1 {
		t.Fatalf("rating summary = %v/%d; want 4/1", rated.Body.Rating, rated.Body.RatingCount)
	}
	if rated.Body.MyRating == nil || *rated.Body.MyRating != 4 {
		t.Fatalf("my rating = %#v; want 4", rated.Body.MyRating)
	}

	secondCtx := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	if _, err := rateReadingOrder(secondCtx, db, created.Body.ID, 2); err != nil {
		t.Fatalf("rate reading order as second user: %v", err)
	}
	refreshed, err := getReadingOrder(ctx, db, created.Body.ID)
	if err != nil {
		t.Fatalf("get reading order: %v", err)
	}
	if refreshed.Body.Rating != 3 || refreshed.Body.RatingCount != 2 {
		t.Fatalf("rating summary = %v/%d; want 3/2", refreshed.Body.Rating, refreshed.Body.RatingCount)
	}
	if refreshed.Body.MyRating == nil || *refreshed.Body.MyRating != 4 {
		t.Fatalf("my rating after second user = %#v; want 4", refreshed.Body.MyRating)
	}

	cleared, err := rateReadingOrder(ctx, db, created.Body.ID, 0)
	if err != nil {
		t.Fatalf("clear reading order rating: %v", err)
	}
	if cleared.Body.Rating != 2 || cleared.Body.RatingCount != 1 {
		t.Fatalf("rating summary after clear = %v/%d; want 2/1", cleared.Body.Rating, cleared.Body.RatingCount)
	}
	if cleared.Body.MyRating != nil {
		t.Fatalf("my rating after clear = %#v; want nil", cleared.Body.MyRating)
	}
}

func TestStartReadingOrderIsPerUserAndIdempotent(t *testing.T) {
	ctx := testUserContext()
	db := setupReadingOrderCBLTestDB(t)
	if _, err := db.Exec(`INSERT INTO users (id, name) VALUES (2, 'Second Reader')`); err != nil {
		t.Fatalf("insert second user: %v", err)
	}

	created, err := createReadingOrder(ctx, db, nil, ReadingOrderPayload{Name: "Started order"})
	if err != nil {
		t.Fatalf("create reading order: %v", err)
	}
	started, err := startReadingOrder(ctx, db, created.Body.ID)
	if err != nil {
		t.Fatalf("start reading order: %v", err)
	}
	if started.Body.StartedAt == nil || *started.Body.StartedAt == "" {
		t.Fatal("startedAt is empty after starting reading order")
	}
	firstStartedAt := *started.Body.StartedAt
	startedList, err := listReadingOrders(ctx, db, &ReadingOrderListInput{Started: "true"})
	if err != nil {
		t.Fatalf("list started reading orders: %v", err)
	}
	if len(startedList.Body) != 1 || startedList.Body[0].ID != created.Body.ID {
		t.Fatalf("started reading orders = %#v; want created order", startedList.Body)
	}

	startedAgain, err := startReadingOrder(ctx, db, created.Body.ID)
	if err != nil {
		t.Fatalf("start reading order again: %v", err)
	}
	if startedAgain.Body.StartedAt == nil || *startedAgain.Body.StartedAt != firstStartedAt {
		t.Fatalf("repeated start changed startedAt from %q to %#v", firstStartedAt, startedAgain.Body.StartedAt)
	}

	secondCtx := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	secondDetail, err := getReadingOrder(secondCtx, db, created.Body.ID)
	if err != nil {
		t.Fatalf("get reading order as second user: %v", err)
	}
	if secondDetail.Body.StartedAt != nil {
		t.Fatalf("second user's startedAt = %#v; want nil", secondDetail.Body.StartedAt)
	}

	stopped, err := stopReadingOrder(ctx, db, created.Body.ID)
	if err != nil {
		t.Fatalf("stop reading order: %v", err)
	}
	if stopped.Body.StartedAt != nil {
		t.Fatalf("startedAt after stop = %#v; want nil", stopped.Body.StartedAt)
	}
}
