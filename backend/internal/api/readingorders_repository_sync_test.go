package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

func TestValidateCBLRepositorySyncSettingsNormalizesRepositories(t *testing.T) {
	settings := CBLRepositorySyncSettings{
		Repositories: []string{
			"https://github.com/DieselTech/CBL-ReadingLists/",
			"https://github.com/DieselTech/CBL-ReadingLists.git",
			"https://github.com/example/other",
		},
		Folders: []CBLRepositoryFolderSelection{
			{RepositoryURL: "https://github.com/DieselTech/CBL-ReadingLists.git", Path: "/Marvel/Events/"},
			{RepositoryURL: "https://github.com/DieselTech/CBL-ReadingLists", Path: "Marvel/Events/Secret Wars"},
			{RepositoryURL: "https://github.com/example/other", Path: "DC"},
		},
		Schedule:  "weekly",
		Weekdays:  []string{"Friday", "friday"},
		StartTime: "04:30",
		AutoSync:  true,
	}
	if err := validateCBLRepositorySyncSettings(&settings); err != nil {
		t.Fatalf("validate settings: %v", err)
	}
	if len(settings.Repositories) != 2 || settings.Repositories[0] != "https://github.com/DieselTech/CBL-ReadingLists" {
		t.Fatalf("repositories = %#v; want canonical unique URLs", settings.Repositories)
	}
	if len(settings.Weekdays) != 1 || settings.Weekdays[0] != "friday" {
		t.Fatalf("weekdays = %#v; want one normalized Friday", settings.Weekdays)
	}
	if len(settings.Folders) != 2 || settings.Folders[0].Path != "Marvel/Events" || settings.Folders[1].Path != "DC" {
		t.Fatalf("folders = %#v; want canonical folders without redundant descendants", settings.Folders)
	}
}

func TestValidateCBLRepositorySyncSettingsRejectsFolderFromUnconfiguredRepository(t *testing.T) {
	settings := defaultCBLRepositorySyncSettings()
	settings.Folders = []CBLRepositoryFolderSelection{{
		RepositoryURL: "https://github.com/example/other",
		Path:          "Marvel",
	}}
	if err := validateCBLRepositorySyncSettings(&settings); err == nil {
		t.Fatal("validate settings succeeded; want unconfigured folder repository rejected")
	}
}

func TestCBLRepositorySyncSubscriptionSendsSnapshotAndProgress(t *testing.T) {
	db := newMetronComicScannerTestDB(t)
	syncer := NewCBLRepositorySyncer(db, nil, nil)
	updates, unsubscribe := syncer.subscribe(context.Background())
	defer unsubscribe()

	initial := <-updates
	if initial.Imported != 0 || initial.CurrentFile != "" {
		t.Fatalf("initial status = %+v; want idle snapshot", initial)
	}

	syncer.setCurrentFile("Marvel/Events/Secret Wars.cbl")
	progress := <-updates
	if progress.CurrentFile != "Marvel/Events/Secret Wars.cbl" {
		t.Fatalf("progress current file = %q; want SSE subscriber update", progress.CurrentFile)
	}

	syncer.increment("imported")
	progress = <-updates
	if progress.Imported != 1 {
		t.Fatalf("progress imported = %d; want SSE subscriber update", progress.Imported)
	}
}

func TestCBLRepositoryFilesInFoldersIncludesDescendantsOnly(t *testing.T) {
	files := []cblGitHubFile{
		{Path: "DC/Batman/Year One.cbl"},
		{Path: "DC/Batman Beyond/Neo Year.cbl"},
		{Path: "DC/Batman/Modern/Court of Owls.cbl"},
		{Path: "Marvel/Events/Secret Wars.cbl"},
	}
	filtered := cblRepositoryFilesInFolders(files, []string{"DC/Batman", "Marvel/Events"})
	if len(filtered) != 3 || filtered[0].Path != "DC/Batman/Year One.cbl" || filtered[1].Path != "DC/Batman/Modern/Court of Owls.cbl" || filtered[2].Path != "Marvel/Events/Secret Wars.cbl" {
		t.Fatalf("filtered files = %#v; want only files inside selected folder boundaries", filtered)
	}
}

func TestCBLRepositoryPartRecognizesDescriptiveAndAbbreviatedParts(t *testing.T) {
	tests := []struct {
		path       string
		parentName string
		part       int
	}{
		{"DC/Batman/[DC] Batman Modern Age - Part 01 Year One.cbl", "[DC] Batman Modern Age", 1},
		{"Marvel/Guardians/Guardians of the Galaxy (2008) pt.2.cbl", "Guardians of the Galaxy (2008)", 2},
	}
	for _, test := range tests {
		_, parentName, part, ok := cblRepositoryPart(test.path)
		if !ok || parentName != test.parentName || part != test.part {
			t.Fatalf("cblRepositoryPart(%q) = %q, %d, %v; want %q, %d, true", test.path, parentName, part, ok, test.parentName, test.part)
		}
	}
	if _, _, _, ok := cblRepositoryPart("DC/Batman/Batman 001 - Golden Age.cbl"); ok {
		t.Fatal("ordinary numbered list was treated as multipart")
	}
}

func TestSelectedRepositoryFilesKeepsSingleMultipartPart(t *testing.T) {
	files := []cblGitHubFile{
		{Path: "Test Saga - Part 01.cbl", SHA: "one"},
		{Path: "Test Saga - Part 02.cbl", SHA: "two"},
	}
	selected, err := selectedRepositoryFiles(files, map[string]bool{"Test Saga - Part 01.cbl": true}, "https://github.com/example/lists")
	if err != nil {
		t.Fatalf("selectedRepositoryFiles: %v", err)
	}
	if len(selected) != 1 || selected[0].Path != "Test Saga - Part 01.cbl" {
		t.Fatalf("selected files = %#v; want only explicitly selected part", selected)
	}
}

func TestCBLRepositorySyncUpdatesChangedMultipartFileInOneOrder(t *testing.T) {
	db := setupReadingOrderCBLTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE app_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		CREATE TABLE cbl_repository_files (
			repository_url TEXT NOT NULL,
			file_path TEXT NOT NULL,
			content_sha TEXT NOT NULL,
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			group_key TEXT NOT NULL DEFAULT '',
			imported_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (repository_url, file_path)
		);
	`); err != nil {
		t.Fatalf("create repository sync schema: %v", err)
	}

	var mu sync.Mutex
	partOneSHA := "sha-one"
	partOneIssue := "1"
	partOnePath := "DC/Test/Test Saga - Part 01 Beginning.cbl"
	partTwoPath := "DC/Test/Test Saga - Part 02 Finale.cbl"
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		mu.Lock()
		sha := partOneSHA
		issue := partOneIssue
		mu.Unlock()
		switch request.URL.Path {
		case "/repos/owner/lists":
			fmt.Fprint(response, `{"default_branch":"main"}`)
		case "/repos/owner/lists/git/trees/main":
			fmt.Fprintf(response, `{"truncated":false,"tree":[`+
				`{"path":%q,"type":"blob","sha":%q,"size":300},`+
				`{"path":%q,"type":"blob","sha":"sha-two","size":300}]}`, partOnePath, sha, partTwoPath)
		case "/owner/lists/main/" + partOnePath:
			fmt.Fprintf(response, `<ReadingList><Name>Test Saga - Part 01 Beginning</Name><Books><Book Series="Series" Number="%s" Volume="2020" /></Books></ReadingList>`, issue)
		case "/owner/lists/main/" + partTwoPath:
			fmt.Fprint(response, `<ReadingList><Name>Test Saga - Part 02 Finale</Name><Books><Book Series="Series" Number="2" Volume="2020" /></Books></ReadingList>`)
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	syncer := NewCBLRepositorySyncer(db, nil, nil)
	syncer.httpClient = server.Client()
	syncer.apiBase = server.URL
	syncer.rawBase = server.URL
	repository, _ := parseCBLGitHubRepository("https://github.com/owner/lists")
	ctx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	catalog, err := syncer.availableRepositoryFiles(ctx, CBLRepositorySyncSettings{Repositories: []string{repository.URL}})
	if err != nil {
		t.Fatalf("list available files: %v", err)
	}
	if len(catalog) != 2 || catalog[0].MultipartGroup != "Test Saga" || catalog[1].MultipartGroup != "Test Saga" {
		t.Fatalf("catalog = %#v; want both files marked as one multipart group", catalog)
	}
	if err := syncer.syncRepository(ctx, repository, map[string]bool{partOnePath: true, partTwoPath: true}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if status := syncer.snapshot(ctx); status.FilesFound != 2 {
		t.Fatalf("selected multipart files found = %d; want both explicitly selected parts", status.FilesFound)
	}

	var originalOrderID int
	if err := db.Get(&originalOrderID, `SELECT reading_order_id FROM cbl_repository_files WHERE repository_url = ? AND file_path = ?`, repository.URL, partOnePath); err != nil {
		t.Fatalf("read first mapping: %v", err)
	}
	var orderCount int
	if err := db.Get(&orderCount, `SELECT COUNT(*) FROM reading_orders`); err != nil || orderCount != 1 {
		t.Fatalf("reading order count = %d, %v; want one combined multipart order", orderCount, err)
	}
	var mappedOrderIDs []int
	if err := db.Select(&mappedOrderIDs, `SELECT reading_order_id FROM cbl_repository_files ORDER BY file_path`); err != nil {
		t.Fatalf("read multipart mappings: %v", err)
	}
	if len(mappedOrderIDs) != 2 || mappedOrderIDs[0] != originalOrderID || mappedOrderIDs[1] != originalOrderID {
		t.Fatalf("multipart mapping IDs = %#v; want both files mapped to order %d", mappedOrderIDs, originalOrderID)
	}

	mu.Lock()
	partOneSHA = "sha-one-updated"
	partOneIssue = "3"
	mu.Unlock()
	if err := syncer.syncRepository(ctx, repository, nil); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	var updatedOrderID int
	if err := db.Get(&updatedOrderID, `SELECT reading_order_id FROM cbl_repository_files WHERE repository_url = ? AND file_path = ?`, repository.URL, partOnePath); err != nil {
		t.Fatalf("read updated mapping: %v", err)
	}
	if updatedOrderID != originalOrderID {
		t.Fatalf("changed file reading order id = %d; want existing id %d", updatedOrderID, originalOrderID)
	}
	var issue string
	if err := db.Get(&issue, `SELECT c.issue FROM reading_order_comics roc JOIN comics c ON c.id = roc.comic_id WHERE roc.reading_order_id = ? ORDER BY roc.position LIMIT 1`, originalOrderID); err != nil {
		t.Fatalf("read updated part entry: %v", err)
	}
	if strings.TrimSpace(issue) != "3" {
		t.Fatalf("updated part issue = %q; want 3", issue)
	}
	if err := db.Get(&orderCount, `SELECT COUNT(*) FROM reading_orders`); err != nil || orderCount != 1 {
		t.Fatalf("reading order count after update = %d, %v; want no duplicate orders", orderCount, err)
	}
	var sectionCount int
	if err := db.Get(&sectionCount, `SELECT COUNT(*) FROM reading_order_sections WHERE reading_order_id = ?`, originalOrderID); err != nil || sectionCount != 2 {
		t.Fatalf("multipart section count = %d, %v; want one section per part", sectionCount, err)
	}
}

func TestCBLRepositoryMetronResolverPrefersComicVineID(t *testing.T) {
	db := newMetronImportTestDB(t)
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.URL.String())
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/issue/":
			fmt.Fprint(response, `{"results":[{"id":55,"cv_id":987654,"series":{"name":"Saga","year_began":2012},"number":"1","publisher":{"name":"Image"}}]}`)
		case "/issue/55/":
			fmt.Fprint(response, `{"id":55,"cv_id":987654,"series":{"name":"Saga","year_began":2012},"number":"1","publisher":{"name":"Image"},"cover_date":"2012-03-01"}`)
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	syncer := NewCBLRepositorySyncer(db, metron.New(metron.Config{BaseURL: server.URL}), nil)
	comic, err := syncer.resolveMissingComicFromMetron(testUserContext(), cblBook{
		Series:    "Saga",
		Number:    "1",
		Databases: []cblDatabase{{Name: "comicvine", Issue: "987654"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if comic == nil || comic.MetronIssueID == nil || *comic.MetronIssueID != 55 {
		t.Fatalf("comic = %#v; want imported Metron issue 55", comic)
	}
	if len(requests) != 2 || requests[0] != "/issue/?cv_id=987654" || requests[1] != "/issue/55/" {
		t.Fatalf("Metron requests = %#v; want Comic Vine search followed by detail", requests)
	}
}

func TestCBLRepositoryMetronResolverFallsBackToNameAndWaitsForSelection(t *testing.T) {
	db := newMetronImportTestDB(t)
	var requests []string
	var requestsMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestsMu.Lock()
		requests = append(requests, request.URL.String())
		requestsMu.Unlock()
		response.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/issue/" && request.URL.Query().Get("cv_id") != "":
			fmt.Fprint(response, `{"results":[]}`)
		case request.URL.Path == "/issue/":
			fmt.Fprint(response, `{"results":[`+
				`{"id":71,"series":{"name":"The Example","year_began":2001},"number":"4","publisher":{"name":"First"}},`+
				`{"id":72,"series":{"name":"The Example","year_began":2020},"number":"4","publisher":{"name":"Second"}}]}`)
		case request.URL.Path == "/issue/72/":
			fmt.Fprint(response, `{"id":72,"series":{"name":"The Example","year_began":2020},"number":"4","publisher":{"name":"Second"}}`)
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	syncer := NewCBLRepositorySyncer(db, metron.New(metron.Config{BaseURL: server.URL}), nil)
	syncer.status.Running = true
	syncer.resolutionChoices = make(chan cblMetronResolutionChoice, 1)
	type result struct {
		comic *Comic
		err   error
	}
	resultCh := make(chan result, 1)
	go func() {
		comic, err := syncer.resolveMissingComicFromMetron(testUserContext(), cblBook{
			Series:    "The Example",
			Number:    "4",
			Databases: []cblDatabase{{Name: "ComicVine", Issue: "4444"}},
		})
		resultCh <- result{comic: comic, err: err}
	}()

	var pending *CBLMetronIssueResolution
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		syncer.mu.Lock()
		pending = syncer.status.PendingResolution
		syncer.mu.Unlock()
		if pending != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if pending == nil || len(pending.Candidates) != 2 {
		t.Fatalf("pending resolution = %#v; want two Metron candidates", pending)
	}
	if err := syncer.resolveMetronIssue(pending.ID, 72); err != nil {
		t.Fatal(err)
	}

	select {
	case resolved := <-resultCh:
		if resolved.err != nil {
			t.Fatal(resolved.err)
		}
		if resolved.comic == nil || resolved.comic.MetronIssueID == nil || *resolved.comic.MetronIssueID != 72 {
			t.Fatalf("comic = %#v; want selected Metron issue 72", resolved.comic)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("resolver did not continue after selection")
	}

	requestsMu.Lock()
	defer requestsMu.Unlock()
	want := []string{
		"/issue/?cv_id=4444",
		"/issue/?number=4&series_name=The+Example",
		"/issue/72/",
	}
	if len(requests) != len(want) {
		t.Fatalf("Metron requests = %#v; want %#v", requests, want)
	}
	for i := range want {
		if requests[i] != want[i] {
			t.Fatalf("Metron request %d = %q; want %q", i, requests[i], want[i])
		}
	}
}
