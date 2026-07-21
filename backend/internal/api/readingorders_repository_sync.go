package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/jmoiron/sqlx"
)

const (
	cblRepositorySettingsKey = "cbl_repository_sync_settings"
	cblRepositoryLastRunKey  = "cbl_repository_sync_last_scheduled_date"
	cblRepositoryMaxFileSize = 4 << 20
)

type CBLRepositorySyncSettings struct {
	Enabled      bool                           `json:"enabled" doc:"Whether repository imports can run."`
	Repositories []string                       `json:"repositories" doc:"Public GitHub repository URLs containing CBL files."`
	Folders      []CBLRepositoryFolderSelection `json:"folders,omitempty" doc:"Optional repository subfolders to import recursively. When set, only repositories and folders in this list are imported."`
	AutoSync     bool                           `json:"autoSync" doc:"Whether repositories are checked on a schedule."`
	Schedule     string                         `json:"schedule" enum:"daily,weekly"`
	Weekdays     []string                       `json:"weekdays,omitempty"`
	StartTime    string                         `json:"startTime" example:"04:00"`
}

type CBLRepositorySyncStatus struct {
	Settings               CBLRepositorySyncSettings `json:"settings"`
	Running                bool                      `json:"running"`
	StartedAt              string                    `json:"startedAt,omitempty"`
	FinishedAt             string                    `json:"finishedAt,omitempty"`
	StopReason             string                    `json:"stopReason,omitempty"`
	CurrentRepository      string                    `json:"currentRepository,omitempty"`
	CurrentFile            string                    `json:"currentFile,omitempty"`
	RepositoriesProcessed  int                       `json:"repositoriesProcessed"`
	FilesFound             int                       `json:"filesFound"`
	Imported               int                       `json:"imported"`
	Updated                int                       `json:"updated"`
	Unchanged              int                       `json:"unchanged"`
	Failed                 int                       `json:"failed"`
	LastError              string                    `json:"lastError,omitempty"`
	ResolveMissingOnMetron bool                      `json:"resolveMissingOnMetron,omitempty"`
	PendingResolution      *CBLMetronIssueResolution `json:"pendingResolution,omitempty"`
}

type CBLRepositorySyncEvent struct {
	Sync CBLRepositorySyncStatus `json:"sync"`
}

type CBLRepositoryFile struct {
	RepositoryURL  string `json:"repositoryUrl"`
	Path           string `json:"path"`
	Size           int64  `json:"size"`
	MultipartGroup string `json:"multipartGroup,omitempty"`
	Part           int    `json:"part,omitempty"`
}

type CBLRepositoryFileSelection struct {
	RepositoryURL string `json:"repositoryUrl"`
	Path          string `json:"path"`
}

type CBLRepositoryFolderSelection struct {
	RepositoryURL string `json:"repositoryUrl" doc:"Configured GitHub repository URL."`
	Path          string `json:"path" doc:"Repository-relative folder path. CBL files in this folder and its descendants are included." example:"Marvel/Events"`
}

type CBLMetronIssueCandidate struct {
	ID          int    `json:"id"`
	ComicVineID int    `json:"comicVineId,omitempty"`
	Series      string `json:"series"`
	SeriesYear  int    `json:"seriesYear,omitempty"`
	Number      string `json:"number"`
	Publisher   string `json:"publisher,omitempty"`
	CoverDate   string `json:"coverDate,omitempty"`
	CoverImage  string `json:"coverImage,omitempty"`
}

type CBLMetronIssueResolution struct {
	ID          string                    `json:"id"`
	Series      string                    `json:"series"`
	Number      string                    `json:"number"`
	Volume      string                    `json:"volume,omitempty"`
	Year        string                    `json:"year,omitempty"`
	ComicVineID int                       `json:"comicVineId,omitempty"`
	Candidates  []CBLMetronIssueCandidate `json:"candidates"`
}

type cblMetronResolutionChoice struct {
	ResolutionID  string
	MetronIssueID int
}

type cblRepositorySyncer struct {
	db           *sqlx.DB
	metronClient *metron.Client
	covers       *CoverCache
	httpClient   *http.Client
	apiBase      string
	rawBase      string

	mu                sync.Mutex
	status            CBLRepositorySyncStatus
	cancel            context.CancelFunc
	wake              chan struct{}
	shutdown          context.CancelFunc
	nextSubscriberID  uint64
	subscribers       map[uint64]chan CBLRepositorySyncStatus
	resolutionChoices chan cblMetronResolutionChoice
	nextResolutionID  uint64
}

type cblGitHubRepository struct {
	URL   string
	Owner string
	Name  string
}

type cblGitHubFile struct {
	Path string
	SHA  string
	Size int64
	Part int
}

type cblRepositoryFileState struct {
	RepositoryURL  string `db:"repository_url"`
	FilePath       string `db:"file_path"`
	ContentSHA     string `db:"content_sha"`
	ReadingOrderID int    `db:"reading_order_id"`
	GroupKey       string `db:"group_key"`
}

func NewCBLRepositorySyncer(db *sqlx.DB, metronClient *metron.Client, covers *CoverCache) *cblRepositorySyncer {
	return &cblRepositorySyncer{
		db:           db,
		metronClient: metronClient,
		covers:       covers,
		httpClient:   &http.Client{Timeout: 60 * time.Second},
		apiBase:      "https://api.github.com",
		rawBase:      "https://raw.githubusercontent.com",
		wake:         make(chan struct{}, 1),
		subscribers:  map[uint64]chan CBLRepositorySyncStatus{},
	}
}

func (s *cblRepositorySyncer) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.shutdown = cancel
	go s.scheduleLoop(ctx)
}

func (s *cblRepositorySyncer) Stop() {
	if s.shutdown != nil {
		s.shutdown()
	}
	s.stop("server stopped")
}

func (s *cblRepositorySyncer) scheduleLoop(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkSchedule(ctx, time.Now())
		case <-s.wake:
			s.checkSchedule(ctx, time.Now())
		}
	}
}

func (s *cblRepositorySyncer) checkSchedule(ctx context.Context, now time.Time) {
	settings, err := loadCBLRepositorySyncSettings(ctx, s.db)
	if err != nil || !settings.Enabled || !settings.AutoSync || now.Format("15:04") != settings.StartTime {
		return
	}
	if settings.Schedule == "weekly" {
		matches := false
		for _, day := range settings.Weekdays {
			matches = matches || weekdayNames[day] == now.Weekday()
		}
		if !matches {
			return
		}
	}
	date := now.Format("2006-01-02")
	var last string
	_ = s.db.GetContext(ctx, &last, `SELECT value FROM app_settings WHERE key = ?`, cblRepositoryLastRunKey)
	if last == date || s.trigger("scheduled", nil, false) != nil {
		return
	}
	_, _ = s.db.ExecContext(ctx, `INSERT INTO app_settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, cblRepositoryLastRunKey, date)
}

func (s *cblRepositorySyncer) trigger(reason string, selections []CBLRepositoryFileSelection, resolveMissingOnMetron bool) error {
	settings, err := loadCBLRepositorySyncSettings(context.Background(), s.db)
	if err != nil {
		return err
	}
	if !settings.Enabled {
		return errors.New("CBL repository importing is disabled")
	}
	selectedFiles, err := selectedCBLRepositoryFiles(settings, selections)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		return errors.New("CBL repository import is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.resolutionChoices = make(chan cblMetronResolutionChoice, 1)
	s.status = CBLRepositorySyncStatus{
		Settings:               settings,
		Running:                true,
		StartedAt:              currentTimestamp(),
		StopReason:             reason,
		ResolveMissingOnMetron: resolveMissingOnMetron,
	}
	s.mu.Unlock()
	s.broadcast()
	go s.run(ctx, settings, selectedFiles, resolveMissingOnMetron)
	return nil
}

func selectedCBLRepositoryFiles(settings CBLRepositorySyncSettings, selections []CBLRepositoryFileSelection) (map[string]map[string]bool, error) {
	if len(selections) == 0 {
		return nil, nil
	}
	configured := map[string]string{}
	for _, value := range settings.Repositories {
		repository, err := parseCBLGitHubRepository(value)
		if err != nil {
			return nil, err
		}
		configured[strings.ToLower(repository.URL)] = repository.URL
	}
	selected := map[string]map[string]bool{}
	for _, selection := range selections {
		repository, err := parseCBLGitHubRepository(selection.RepositoryURL)
		if err != nil {
			return nil, err
		}
		configuredURL, ok := configured[strings.ToLower(repository.URL)]
		if !ok {
			return nil, fmt.Errorf("repository %q is not configured", repository.URL)
		}
		filePath := strings.TrimSpace(selection.Path)
		if filePath == "" || !strings.EqualFold(path.Ext(filePath), ".cbl") {
			return nil, fmt.Errorf("selected path %q is not a CBL file", filePath)
		}
		if selected[configuredURL] == nil {
			selected[configuredURL] = map[string]bool{}
		}
		selected[configuredURL][filePath] = true
	}
	return selected, nil
}

func (s *cblRepositorySyncer) stop(reason string) bool {
	s.mu.Lock()
	if !s.status.Running || s.cancel == nil {
		s.mu.Unlock()
		return false
	}
	s.status.StopReason = reason
	s.cancel()
	s.mu.Unlock()
	s.broadcast()
	return true
}

func (s *cblRepositorySyncer) run(ctx context.Context, settings CBLRepositorySyncSettings, selectedFiles map[string]map[string]bool, resolveMissingOnMetron bool) {
	userID, err := ensureDefaultUser(ctx, s.db)
	if err == nil {
		ctx = context.WithValue(ctx, contextUserIDKey{}, userID)
		resolveMissing := cblMissingComicResolver(nil)
		if resolveMissingOnMetron {
			resolveMissing = s.resolveMissingComicFromMetron
		}
		for _, value := range settings.Repositories {
			if ctx.Err() != nil {
				break
			}
			repository, parseErr := parseCBLGitHubRepository(value)
			if parseErr != nil {
				s.recordFailure(parseErr)
				continue
			}
			selection := selectedFiles[repository.URL]
			if selectedFiles != nil && len(selection) == 0 {
				continue
			}
			folderPaths := cblRepositoryFolderPaths(settings, repository.URL)
			if selectedFiles != nil {
				folderPaths = nil
			} else if len(settings.Folders) > 0 && len(folderPaths) == 0 {
				continue
			}
			s.update(func(status *CBLRepositorySyncStatus) {
				status.CurrentRepository = repository.URL
				status.CurrentFile = ""
			})
			if syncErr := s.syncRepositoryWithResolver(ctx, repository, selection, folderPaths, resolveMissing); syncErr != nil {
				s.recordFailure(syncErr)
			}
			s.update(func(status *CBLRepositorySyncStatus) { status.RepositoriesProcessed++ })
		}
	}
	if err != nil {
		s.recordFailure(err)
	}
	s.mu.Lock()
	s.status.Running = false
	s.status.FinishedAt = currentTimestamp()
	s.status.CurrentRepository = ""
	s.status.CurrentFile = ""
	s.status.PendingResolution = nil
	if ctx.Err() != nil {
		if s.status.StopReason == "manual" || s.status.StopReason == "scheduled" {
			s.status.StopReason = "stopped"
		}
	} else if s.status.Failed > 0 {
		s.status.StopReason = "complete with errors"
	} else {
		s.status.StopReason = "complete"
	}
	s.cancel = nil
	s.resolutionChoices = nil
	s.mu.Unlock()
	s.broadcast()
}

func cblRepositoryFolderPaths(settings CBLRepositorySyncSettings, repositoryURL string) []string {
	folders := make([]string, 0)
	for _, selection := range settings.Folders {
		if strings.EqualFold(selection.RepositoryURL, repositoryURL) {
			folders = append(folders, selection.Path)
		}
	}
	return folders
}

func (s *cblRepositorySyncer) syncRepository(ctx context.Context, repository cblGitHubRepository, selectedPaths map[string]bool) error {
	return s.syncRepositoryWithResolver(ctx, repository, selectedPaths, nil, nil)
}

func (s *cblRepositorySyncer) syncRepositoryWithResolver(ctx context.Context, repository cblGitHubRepository, selectedPaths map[string]bool, folderPaths []string, resolveMissing cblMissingComicResolver) error {
	branch, files, err := s.listRepositoryFiles(ctx, repository)
	if err != nil {
		return err
	}
	states, err := s.repositoryFileStates(ctx, repository.URL)
	if err != nil {
		return err
	}
	if selectedPaths == nil {
		files = cblRepositoryFilesInFolders(files, folderPaths)
	}
	selectedPaths = expandManagedMultipartSelections(files, selectedPaths, states)
	files, err = selectedRepositoryFiles(files, selectedPaths, repository.URL)
	if err != nil {
		return err
	}
	s.update(func(status *CBLRepositorySyncStatus) { status.FilesFound += len(files) })

	groups := map[string][]cblGitHubFile{}
	parentNames := map[string]string{}
	singles := make([]cblGitHubFile, 0)
	for _, file := range files {
		groupKey, parentName, partNumber, ok := cblRepositoryPart(file.Path)
		if !ok {
			singles = append(singles, file)
			continue
		}
		file.Part = partNumber
		groups[groupKey] = append(groups[groupKey], file)
		parentNames[groupKey] = parentName
	}
	for key, group := range groups {
		if len(group) < 2 {
			singles = append(singles, group...)
			delete(groups, key)
		}
	}
	sort.Slice(singles, func(i, j int) bool { return strings.ToLower(singles[i].Path) < strings.ToLower(singles[j].Path) })
	for _, file := range singles {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		s.syncSingleFile(ctx, repository, branch, file, states[file.Path], resolveMissing)
	}
	groupKeys := make([]string, 0, len(groups))
	for key := range groups {
		groupKeys = append(groupKeys, key)
	}
	sort.Strings(groupKeys)
	for _, key := range groupKeys {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		group := groups[key]
		sort.SliceStable(group, func(i, j int) bool {
			if group[i].Part != group[j].Part {
				return group[i].Part < group[j].Part
			}
			return strings.ToLower(group[i].Path) < strings.ToLower(group[j].Path)
		})
		s.syncMultipartGroup(ctx, repository, branch, key, parentNames[key], group, states, resolveMissing)
	}
	return nil
}

func cblRepositoryFilesInFolders(files []cblGitHubFile, folderPaths []string) []cblGitHubFile {
	if len(folderPaths) == 0 {
		return files
	}
	filtered := make([]cblGitHubFile, 0, len(files))
	for _, file := range files {
		for _, folderPath := range folderPaths {
			if pathIsInCBLRepositoryFolder(file.Path, folderPath) {
				filtered = append(filtered, file)
				break
			}
		}
	}
	return filtered
}

func pathIsInCBLRepositoryFolder(filePath, folderPath string) bool {
	return filePath == folderPath || strings.HasPrefix(filePath, folderPath+"/")
}

func expandManagedMultipartSelections(files []cblGitHubFile, selectedPaths map[string]bool, states map[string]cblRepositoryFileState) map[string]bool {
	if selectedPaths == nil {
		return nil
	}
	expanded := make(map[string]bool, len(selectedPaths))
	for filePath := range selectedPaths {
		expanded[filePath] = true
		groupKey := states[filePath].GroupKey
		if groupKey == "" {
			continue
		}
		for _, file := range files {
			if candidateKey, _, _, ok := cblRepositoryPart(file.Path); ok && candidateKey == groupKey {
				expanded[file.Path] = true
			}
		}
	}
	return expanded
}

func selectedRepositoryFiles(files []cblGitHubFile, selectedPaths map[string]bool, repositoryURL string) ([]cblGitHubFile, error) {
	if selectedPaths == nil {
		return files, nil
	}
	available := make(map[string]cblGitHubFile, len(files))
	for _, file := range files {
		available[file.Path] = file
	}
	selected := make([]cblGitHubFile, 0, len(selectedPaths))
	for filePath := range selectedPaths {
		file, ok := available[filePath]
		if !ok {
			return nil, fmt.Errorf("selected CBL file %q was not found in %s", filePath, repositoryURL)
		}
		selected = append(selected, file)
	}
	return selected, nil
}

func (s *cblRepositorySyncer) syncSingleFile(ctx context.Context, repository cblGitHubRepository, branch string, file cblGitHubFile, state cblRepositoryFileState, resolveMissing cblMissingComicResolver) {
	s.setCurrentFile(file.Path)
	if state.ReadingOrderID > 0 && state.ContentSHA == file.SHA {
		s.increment("unchanged")
		return
	}
	document, err := s.fetchCBLDocument(ctx, repository, branch, file.Path)
	if err != nil {
		s.recordFailure(err)
		return
	}
	readingOrderID := state.ReadingOrderID
	if readingOrderID > 0 {
		if _, err = updateCBLDocumentWithResolver(ctx, s.db, readingOrderID, document, resolveMissing); err == nil {
			s.increment("updated")
		}
	} else {
		var result cblImportResult
		result, err = importCBLDocumentWithResolver(ctx, s.db, document, resolveMissing)
		if err == nil {
			readingOrderID = result.readingOrder.ID
			s.increment("imported")
		}
	}
	if err != nil {
		s.recordFailure(fmt.Errorf("%s: %w", file.Path, err))
		return
	}
	if err := s.saveRepositoryFileState(ctx, repository.URL, file, readingOrderID, state.GroupKey); err != nil {
		s.recordFailure(err)
	}
}

func (s *cblRepositorySyncer) syncMultipartGroup(ctx context.Context, repository cblGitHubRepository, branch, groupKey, parentName string, files []cblGitHubFile, states map[string]cblRepositoryFileState, resolveMissing cblMissingComicResolver) {
	readingOrderID := 0
	changed := false
	legacyOrderIDs := map[int]bool{}
	for _, file := range files {
		state := states[file.Path]
		if readingOrderID == 0 && state.ReadingOrderID > 0 {
			readingOrderID = state.ReadingOrderID
		}
		if state.ReadingOrderID > 0 {
			legacyOrderIDs[state.ReadingOrderID] = true
		}
		if state.ReadingOrderID == 0 || state.ContentSHA != file.SHA {
			changed = true
		}
	}
	if readingOrderID > 0 {
		for _, file := range files {
			if states[file.Path].ReadingOrderID != readingOrderID || states[file.Path].GroupKey != groupKey {
				changed = true
				break
			}
		}
	}
	if !changed && readingOrderID > 0 {
		for range files {
			s.increment("unchanged")
		}
		return
	}

	documents := make([]cblImportDocument, 0, len(files))
	for _, file := range files {
		if ctx.Err() != nil {
			return
		}
		s.setCurrentFile(file.Path)
		document, err := s.fetchCBLDocument(ctx, repository, branch, file.Path)
		if err != nil {
			s.recordFailure(fmt.Errorf("%s: %w", file.Path, err))
			return
		}
		documents = append(documents, document)
	}
	result, err := importCombinedCBLDocumentsWithResolver(ctx, s.db, readingOrderID, parentName, documents, resolveMissing)
	if err != nil {
		s.recordFailure(err)
		return
	}
	if readingOrderID == 0 {
		s.increment("imported")
	} else {
		s.increment("updated")
	}
	readingOrderID = result.readingOrder.ID
	for _, file := range files {
		if err := s.saveRepositoryFileState(ctx, repository.URL, file, readingOrderID, groupKey); err != nil {
			s.recordFailure(err)
		}
	}
	for legacyID := range legacyOrderIDs {
		if legacyID == readingOrderID {
			continue
		}
		cleanupCBLReadingOrders(ctx, s.db, []int{legacyID})
	}
}
