package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"
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

func defaultCBLRepositorySyncSettings() CBLRepositorySyncSettings {
	return CBLRepositorySyncSettings{
		Repositories: []string{"https://github.com/DieselTech/CBL-ReadingLists"},
		Schedule:     "daily",
		Weekdays:     []string{"monday"},
		StartTime:    "04:00",
	}
}

func loadCBLRepositorySyncSettings(ctx context.Context, db *sqlx.DB) (CBLRepositorySyncSettings, error) {
	settings := defaultCBLRepositorySyncSettings()
	var value string
	if err := db.GetContext(ctx, &value, `SELECT value FROM app_settings WHERE key = ?`, cblRepositorySettingsKey); err != nil {
		if err == sql.ErrNoRows {
			return settings, nil
		}
		return settings, err
	}
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return settings, err
	}
	return settings, nil
}

func validateCBLRepositorySyncSettings(settings *CBLRepositorySyncSettings) error {
	if settings == nil {
		return errors.New("settings are required")
	}
	settings.Schedule = strings.ToLower(strings.TrimSpace(settings.Schedule))
	if settings.Schedule != "daily" && settings.Schedule != "weekly" {
		return errors.New("schedule must be daily or weekly")
	}
	if _, err := time.Parse("15:04", settings.StartTime); err != nil {
		return errors.New("startTime must use HH:MM")
	}
	if len(settings.Repositories) == 0 {
		return errors.New("at least one repository is required")
	}
	seenRepositories := map[string]bool{}
	configuredRepositories := map[string]string{}
	repositories := make([]string, 0, len(settings.Repositories))
	for _, value := range settings.Repositories {
		repository, err := parseCBLGitHubRepository(value)
		if err != nil {
			return err
		}
		key := strings.ToLower(repository.URL)
		if !seenRepositories[key] {
			seenRepositories[key] = true
			repositories = append(repositories, repository.URL)
		}
		configuredRepositories[key] = repository.URL
	}
	settings.Repositories = repositories
	folders := make([]CBLRepositoryFolderSelection, 0, len(settings.Folders))
	seenFolders := map[string]bool{}
	for _, selection := range settings.Folders {
		repository, err := parseCBLGitHubRepository(selection.RepositoryURL)
		if err != nil {
			return err
		}
		repositoryURL, ok := configuredRepositories[strings.ToLower(repository.URL)]
		if !ok {
			return fmt.Errorf("folder repository %q is not configured", repository.URL)
		}
		folderPath, err := normalizeCBLRepositoryFolder(selection.Path)
		if err != nil {
			return err
		}
		key := strings.ToLower(repositoryURL) + "\n" + folderPath
		if seenFolders[key] {
			continue
		}
		seenFolders[key] = true
		folders = append(folders, CBLRepositoryFolderSelection{RepositoryURL: repositoryURL, Path: folderPath})
	}
	sort.Slice(folders, func(i, j int) bool {
		if !strings.EqualFold(folders[i].RepositoryURL, folders[j].RepositoryURL) {
			return strings.ToLower(folders[i].RepositoryURL) < strings.ToLower(folders[j].RepositoryURL)
		}
		return strings.ToLower(folders[i].Path) < strings.ToLower(folders[j].Path)
	})
	settings.Folders = removeRedundantCBLRepositoryFolders(folders)
	seenDays := map[string]bool{}
	days := make([]string, 0, len(settings.Weekdays))
	for _, value := range settings.Weekdays {
		day := strings.ToLower(strings.TrimSpace(value))
		if _, ok := weekdayNames[day]; !ok {
			return fmt.Errorf("invalid weekday %q", day)
		}
		if !seenDays[day] {
			seenDays[day] = true
			days = append(days, day)
		}
	}
	sort.Strings(days)
	settings.Weekdays = days
	if settings.AutoSync && settings.Schedule == "weekly" && len(days) == 0 {
		return errors.New("weekly schedules need at least one weekday")
	}
	return nil
}

func normalizeCBLRepositoryFolder(value string) (string, error) {
	value = strings.Trim(strings.TrimSpace(value), "/")
	if value == "" || strings.Contains(value, "\\") {
		return "", fmt.Errorf("folder path %q must be a repository-relative subfolder", value)
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("folder path %q must be a repository-relative subfolder", value)
	}
	return cleaned, nil
}

func removeRedundantCBLRepositoryFolders(folders []CBLRepositoryFolderSelection) []CBLRepositoryFolderSelection {
	filtered := make([]CBLRepositoryFolderSelection, 0, len(folders))
	for _, folder := range folders {
		redundant := false
		for _, selected := range filtered {
			if strings.EqualFold(selected.RepositoryURL, folder.RepositoryURL) && pathIsInCBLRepositoryFolder(folder.Path, selected.Path) {
				redundant = true
				break
			}
		}
		if !redundant {
			filtered = append(filtered, folder)
		}
	}
	return filtered
}

func parseCBLGitHubRepository(value string) (cblGitHubRepository, error) {
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Hostname(), "github.com") {
		return cblGitHubRepository{}, fmt.Errorf("repository %q must be an https://github.com/owner/repository URL", value)
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return cblGitHubRepository{}, fmt.Errorf("repository %q must identify one GitHub repository", value)
	}
	name := strings.TrimSuffix(parts[1], ".git")
	if name == "" || strings.ContainsAny(parts[0]+name, "?#") {
		return cblGitHubRepository{}, fmt.Errorf("repository %q is invalid", value)
	}
	return cblGitHubRepository{
		URL:   "https://github.com/" + parts[0] + "/" + name,
		Owner: parts[0],
		Name:  name,
	}, nil
}

func saveCBLRepositorySyncSettings(ctx context.Context, db *sqlx.DB, settings CBLRepositorySyncSettings) error {
	value, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO app_settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, cblRepositorySettingsKey, string(value))
	return err
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

func (s *cblRepositorySyncer) resolveMissingComicFromMetron(ctx context.Context, book cblBook) (*Comic, error) {
	if s.metronClient == nil {
		return nil, errors.New("metron is not configured")
	}

	var candidates []metron.Issue
	comicVineID, hasComicVineID := cblComicVineID(book)
	var err error
	if hasComicVineID {
		candidates, err = s.metronClient.SearchIssuesByComicVineID(ctx, comicVineID)
		if err != nil {
			return nil, fmt.Errorf("search Metron by Comic Vine ID %d: %w", comicVineID, err)
		}
	}
	if len(candidates) == 0 {
		series := strings.TrimSpace(book.Series)
		number := strings.TrimSpace(book.Number)
		if series == "" || number == "" {
			return nil, nil
		}
		candidates, err = s.metronClient.SearchIssues(ctx, "", series, number)
		if err != nil {
			return nil, fmt.Errorf("search Metron for %s #%s: %w", series, number, err)
		}
	}
	candidates = uniqueMetronIssueCandidates(candidates)
	if len(candidates) == 0 {
		return nil, nil
	}

	selected := candidates[0]
	if len(candidates) > 1 {
		selected, err = s.waitForMetronIssueResolution(ctx, book, comicVineID, candidates)
		if err != nil {
			return nil, err
		}
		if selected.ID == 0 {
			return nil, nil
		}
	}

	detail, err := s.metronClient.GetIssue(ctx, selected.ID)
	if err != nil {
		return nil, fmt.Errorf("load Metron issue %d: %w", selected.ID, err)
	}
	output, err := importMetronComicWithOptions(ctx, s.db, s.metronClient, s.covers, *detail, MetronImportOptions{Mode: "quick"})
	if err != nil {
		return nil, err
	}
	comic := output.Body.Comic
	return &comic, nil
}

func uniqueMetronIssueCandidates(issues []metron.Issue) []metron.Issue {
	seen := map[int]bool{}
	unique := make([]metron.Issue, 0, len(issues))
	for _, issue := range issues {
		if issue.ID <= 0 || seen[issue.ID] {
			continue
		}
		seen[issue.ID] = true
		unique = append(unique, issue)
	}
	return unique
}

func (s *cblRepositorySyncer) waitForMetronIssueResolution(ctx context.Context, book cblBook, comicVineID int, candidates []metron.Issue) (metron.Issue, error) {
	byID := make(map[int]metron.Issue, len(candidates))
	items := make([]CBLMetronIssueCandidate, 0, len(candidates))
	for _, issue := range candidates {
		byID[issue.ID] = issue
		number := issue.Issue
		if number == "" {
			number = issue.Number
		}
		items = append(items, CBLMetronIssueCandidate{
			ID:          issue.ID,
			ComicVineID: issue.ComicVineID,
			Series:      issue.Series,
			SeriesYear:  issue.SeriesYear,
			Number:      number,
			Publisher:   issue.Publisher,
			CoverDate:   issue.CoverDate,
			CoverImage:  issue.CoverImage,
		})
	}

	s.mu.Lock()
	s.nextResolutionID++
	resolutionID := fmt.Sprintf("cbl-metron-%d", s.nextResolutionID)
	choices := s.resolutionChoices
	s.status.PendingResolution = &CBLMetronIssueResolution{
		ID:          resolutionID,
		Series:      strings.TrimSpace(book.Series),
		Number:      strings.TrimSpace(book.Number),
		Volume:      strings.TrimSpace(book.Volume),
		Year:        strings.TrimSpace(book.Year),
		ComicVineID: comicVineID,
		Candidates:  items,
	}
	s.mu.Unlock()
	s.broadcast()
	if choices == nil {
		return metron.Issue{}, errors.New("metron issue selection is unavailable")
	}

	for {
		select {
		case <-ctx.Done():
			return metron.Issue{}, ctx.Err()
		case choice := <-choices:
			if choice.ResolutionID != resolutionID {
				continue
			}
			if choice.MetronIssueID == 0 {
				return metron.Issue{}, nil
			}
			issue, ok := byID[choice.MetronIssueID]
			if !ok {
				return metron.Issue{}, errors.New("selected Metron issue is not a candidate")
			}
			return issue, nil
		}
	}
}

func (s *cblRepositorySyncer) resolveMetronIssue(resolutionID string, metronIssueID int) error {
	s.mu.Lock()
	if !s.status.Running || s.status.PendingResolution == nil || s.status.PendingResolution.ID != resolutionID {
		s.mu.Unlock()
		return errors.New("metron issue selection is no longer pending")
	}
	if metronIssueID != 0 {
		valid := false
		for _, candidate := range s.status.PendingResolution.Candidates {
			valid = valid || candidate.ID == metronIssueID
		}
		if !valid {
			s.mu.Unlock()
			return errors.New("selected Metron issue is not a candidate")
		}
	}
	choices := s.resolutionChoices
	s.status.PendingResolution = nil
	s.mu.Unlock()
	if choices == nil {
		return errors.New("metron issue selection is no longer pending")
	}
	choices <- cblMetronResolutionChoice{ResolutionID: resolutionID, MetronIssueID: metronIssueID}
	s.broadcast()
	return nil
}

func cblRepositoryPart(filePath string) (string, string, int, bool) {
	stem := strings.TrimSuffix(path.Base(filePath), path.Ext(filePath))
	parentName, partNumber, ok := cblMultipartPartName(stem)
	if !ok {
		return "", "", 0, false
	}
	directory := strings.ToLower(path.Dir(filePath))
	keyName := strings.ToLower(strings.Join(strings.Fields(parentName), " "))
	return directory + "/" + keyName, parentName, partNumber, true
}

func (s *cblRepositorySyncer) listRepositoryFiles(ctx context.Context, repository cblGitHubRepository) (string, []cblGitHubFile, error) {
	var metadata struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := s.getJSON(ctx, s.apiBase+"/repos/"+url.PathEscape(repository.Owner)+"/"+url.PathEscape(repository.Name), &metadata); err != nil {
		return "", nil, fmt.Errorf("read %s metadata: %w", repository.URL, err)
	}
	if metadata.DefaultBranch == "" {
		return "", nil, errors.New("repository has no default branch")
	}
	var tree struct {
		Truncated bool `json:"truncated"`
		Tree      []struct {
			Path string `json:"path"`
			Type string `json:"type"`
			SHA  string `json:"sha"`
			Size int64  `json:"size"`
		} `json:"tree"`
	}
	treeURL := s.apiBase + "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/git/trees/" + url.PathEscape(metadata.DefaultBranch) + "?recursive=1"
	if err := s.getJSON(ctx, treeURL, &tree); err != nil {
		return "", nil, fmt.Errorf("read %s tree: %w", repository.URL, err)
	}
	if tree.Truncated {
		return "", nil, errors.New("GitHub returned a truncated repository tree")
	}
	files := make([]cblGitHubFile, 0)
	for _, item := range tree.Tree {
		if item.Type == "blob" && strings.EqualFold(path.Ext(item.Path), ".cbl") && item.Size <= cblRepositoryMaxFileSize {
			files = append(files, cblGitHubFile{Path: item.Path, SHA: item.SHA, Size: item.Size})
		}
	}
	return metadata.DefaultBranch, files, nil
}

func (s *cblRepositorySyncer) availableRepositoryFiles(ctx context.Context, settings CBLRepositorySyncSettings) ([]CBLRepositoryFile, error) {
	available := make([]CBLRepositoryFile, 0)
	for _, value := range settings.Repositories {
		repository, err := parseCBLGitHubRepository(value)
		if err != nil {
			return nil, err
		}
		_, files, err := s.listRepositoryFiles(ctx, repository)
		if err != nil {
			return nil, err
		}
		groupCounts := map[string]int{}
		for _, file := range files {
			if groupKey, _, _, ok := cblRepositoryPart(file.Path); ok {
				groupCounts[groupKey]++
			}
		}
		for _, file := range files {
			item := CBLRepositoryFile{RepositoryURL: repository.URL, Path: file.Path, Size: file.Size}
			if groupKey, parentName, partNumber, ok := cblRepositoryPart(file.Path); ok && groupCounts[groupKey] > 1 {
				item.MultipartGroup = parentName
				item.Part = partNumber
			}
			available = append(available, item)
		}
	}
	sort.Slice(available, func(i, j int) bool {
		if available[i].RepositoryURL != available[j].RepositoryURL {
			return strings.ToLower(available[i].RepositoryURL) < strings.ToLower(available[j].RepositoryURL)
		}
		return strings.ToLower(available[i].Path) < strings.ToLower(available[j].Path)
	})
	return available, nil
}

func (s *cblRepositorySyncer) getJSON(ctx context.Context, target string, output any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "ComicHero-CBL-Importer")
	response, err := s.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("GitHub returned %s", response.Status)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 16<<20)).Decode(output)
}

func (s *cblRepositorySyncer) fetchCBLDocument(ctx context.Context, repository cblGitHubRepository, branch, filePath string) (cblImportDocument, error) {
	target, err := url.Parse(s.rawBase)
	if err != nil {
		return cblImportDocument{}, err
	}
	target.Path = path.Join(target.Path, repository.Owner, repository.Name, branch, filePath)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return cblImportDocument{}, err
	}
	request.Header.Set("User-Agent", "ComicHero-CBL-Importer")
	response, err := s.httpClient.Do(request)
	if err != nil {
		return cblImportDocument{}, fmt.Errorf("download %s: %w", filePath, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return cblImportDocument{}, fmt.Errorf("download %s: GitHub returned %s", filePath, response.Status)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, cblRepositoryMaxFileSize+1))
	if err != nil {
		return cblImportDocument{}, err
	}
	if len(content) > cblRepositoryMaxFileSize {
		return cblImportDocument{}, fmt.Errorf("%s exceeds the CBL file size limit", filePath)
	}
	input := &ReadingOrderCBLImportInput{}
	input.Body.Filename = filePath
	input.Body.Content = string(content)
	documents, err := parseCBLImportDocuments(input)
	if err != nil {
		return cblImportDocument{}, err
	}
	return documents[0], nil
}

func (s *cblRepositorySyncer) repositoryFileStates(ctx context.Context, repositoryURL string) (map[string]cblRepositoryFileState, error) {
	rows := []cblRepositoryFileState{}
	if err := s.db.SelectContext(ctx, &rows, `SELECT repository_url, file_path, content_sha, reading_order_id, group_key FROM cbl_repository_files WHERE repository_url = ?`, repositoryURL); err != nil {
		return nil, err
	}
	states := make(map[string]cblRepositoryFileState, len(rows))
	for _, row := range rows {
		states[row.FilePath] = row
	}
	return states, nil
}

func (s *cblRepositorySyncer) saveRepositoryFileState(ctx context.Context, repositoryURL string, file cblGitHubFile, readingOrderID int, groupKey string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cbl_repository_files (repository_url, file_path, content_sha, reading_order_id, group_key, imported_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(repository_url, file_path) DO UPDATE SET
			content_sha = excluded.content_sha,
			reading_order_id = excluded.reading_order_id,
			group_key = excluded.group_key,
			imported_at = excluded.imported_at
	`, repositoryURL, file.Path, file.SHA, readingOrderID, groupKey, currentTimestamp())
	return err
}

func (s *cblRepositorySyncer) setCurrentFile(file string) {
	s.update(func(status *CBLRepositorySyncStatus) { status.CurrentFile = file })
}

func (s *cblRepositorySyncer) increment(kind string) {
	s.update(func(status *CBLRepositorySyncStatus) {
		switch kind {
		case "imported":
			status.Imported++
		case "updated":
			status.Updated++
		default:
			status.Unchanged++
		}
	})
}

func (s *cblRepositorySyncer) recordFailure(err error) {
	if err == nil {
		return
	}
	s.update(func(status *CBLRepositorySyncStatus) {
		status.Failed++
		status.LastError = err.Error()
	})
}

func (s *cblRepositorySyncer) update(change func(*CBLRepositorySyncStatus)) {
	s.mu.Lock()
	change(&s.status)
	s.mu.Unlock()
	s.broadcast()
}

func (s *cblRepositorySyncer) snapshot(ctx context.Context) CBLRepositorySyncStatus {
	settings, _ := loadCBLRepositorySyncSettings(ctx, s.db)
	s.mu.Lock()
	status := s.status
	s.mu.Unlock()
	status.Settings = settings
	return status
}

func (s *cblRepositorySyncer) subscribe(ctx context.Context) (<-chan CBLRepositorySyncStatus, func()) {
	s.mu.Lock()
	s.nextSubscriberID++
	id := s.nextSubscriberID
	updates := make(chan CBLRepositorySyncStatus, 16)
	s.subscribers[id] = updates
	s.mu.Unlock()
	updates <- s.snapshot(ctx)
	return updates, func() {
		s.mu.Lock()
		if current, ok := s.subscribers[id]; ok {
			delete(s.subscribers, id)
			close(current)
		}
		s.mu.Unlock()
	}
}

func (s *cblRepositorySyncer) broadcast() {
	status := s.snapshot(context.Background())
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, updates := range s.subscribers {
		select {
		case updates <- status:
		default:
			select {
			case <-updates:
			default:
			}
			select {
			case updates <- status:
			default:
			}
		}
	}
}

type CBLRepositorySyncStatusOutput struct{ Body CBLRepositorySyncStatus }
type UpdateCBLRepositorySyncSettingsInput struct{ Body CBLRepositorySyncSettings }
type CBLRepositoryFilesOutput struct{ Body []CBLRepositoryFile }
type TriggerCBLRepositorySyncInput struct {
	Body struct {
		Files                  []CBLRepositoryFileSelection `json:"files,omitempty"`
		ResolveMissingOnMetron bool                         `json:"resolveMissingOnMetron,omitempty"`
	}
}
type ResolveCBLMetronIssueInput struct {
	Body struct {
		ResolutionID  string `json:"resolutionId" minLength:"1"`
		MetronIssueID int    `json:"metronIssueId" minimum:"0"`
	}
}

func RegisterCBLRepositorySyncRoutes(api huma.API, db *sqlx.DB, syncer *cblRepositorySyncer) {
	huma.Register(api, huma.Operation{OperationID: "getCBLRepositorySync", Tags: []string{tagReadingOrders}, Summary: "Get CBL repository sync settings and status", Method: http.MethodGet, Path: "/readingOrders/repository-sync", Errors: []int{401, 403, 500}}, func(ctx context.Context, _ *struct{}) (*CBLRepositorySyncStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		return &CBLRepositorySyncStatusOutput{Body: syncer.snapshot(ctx)}, nil
	})
	sse.Register(api, huma.Operation{OperationID: "streamCBLRepositorySync", Tags: []string{tagReadingOrders}, Summary: "Stream CBL repository sync status", Method: http.MethodGet, Path: "/readingOrders/repository-sync/events", Errors: []int{401, 403, 500}}, map[string]any{"sync": CBLRepositorySyncEvent{}}, func(ctx context.Context, _ *struct{}, send sse.Sender) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return
		}
		updates, unsubscribe := syncer.subscribe(ctx)
		defer unsubscribe()
		for {
			select {
			case <-ctx.Done():
				return
			case status, ok := <-updates:
				if !ok || send.Data(CBLRepositorySyncEvent{Sync: status}) != nil {
					return
				}
			}
		}
	})
	huma.Register(api, huma.Operation{OperationID: "updateCBLRepositorySync", Tags: []string{tagReadingOrders}, Summary: "Update CBL repository sync settings", Method: http.MethodPut, Path: "/readingOrders/repository-sync", Errors: []int{400, 401, 403, 500}}, func(ctx context.Context, input *UpdateCBLRepositorySyncSettingsInput) (*CBLRepositorySyncStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		if err := validateCBLRepositorySyncSettings(&input.Body); err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if err := saveCBLRepositorySyncSettings(ctx, db, input.Body); err != nil {
			return nil, huma.Error500InternalServerError("failed to save CBL repository settings")
		}
		select {
		case syncer.wake <- struct{}{}:
		default:
		}
		syncer.broadcast()
		return &CBLRepositorySyncStatusOutput{Body: syncer.snapshot(ctx)}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "listCBLRepositoryFiles", Tags: []string{tagReadingOrders}, Summary: "List available repository CBL files", Method: http.MethodGet, Path: "/readingOrders/repository-sync/files", Errors: []int{401, 403, 500}}, func(ctx context.Context, _ *struct{}) (*CBLRepositoryFilesOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		settings, err := loadCBLRepositorySyncSettings(ctx, db)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to load CBL repository settings")
		}
		files, err := syncer.availableRepositoryFiles(ctx, settings)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return &CBLRepositoryFilesOutput{Body: files}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "triggerCBLRepositorySync", Tags: []string{tagReadingOrders}, Summary: "Import all or selected CBL repository files", Method: http.MethodPost, Path: "/readingOrders/repository-sync/trigger", DefaultStatus: http.StatusAccepted, Errors: []int{400, 401, 403, 409, 500}}, func(ctx context.Context, input *TriggerCBLRepositorySyncInput) (*CBLRepositorySyncStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		if err := syncer.trigger("manual", input.Body.Files, input.Body.ResolveMissingOnMetron); err != nil {
			if strings.Contains(err.Error(), "already running") {
				return nil, huma.Error409Conflict(err.Error())
			}
			return nil, huma.Error400BadRequest(err.Error())
		}
		return &CBLRepositorySyncStatusOutput{Body: syncer.snapshot(ctx)}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "resolveCBLRepositoryMetronIssue", Tags: []string{tagReadingOrders}, Summary: "Choose a Metron issue for an ambiguous CBL book", Method: http.MethodPost, Path: "/readingOrders/repository-sync/resolve", Errors: []int{400, 401, 403, 409, 500}}, func(ctx context.Context, input *ResolveCBLMetronIssueInput) (*CBLRepositorySyncStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		if err := syncer.resolveMetronIssue(strings.TrimSpace(input.Body.ResolutionID), input.Body.MetronIssueID); err != nil {
			return nil, huma.Error409Conflict(err.Error())
		}
		return &CBLRepositorySyncStatusOutput{Body: syncer.snapshot(ctx)}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "stopCBLRepositorySync", Tags: []string{tagReadingOrders}, Summary: "Stop CBL repository import", Method: http.MethodPost, Path: "/readingOrders/repository-sync/stop", Errors: []int{401, 403, 500}}, func(ctx context.Context, _ *struct{}) (*CBLRepositorySyncStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		syncer.stop("stopped by admin")
		return &CBLRepositorySyncStatusOutput{Body: syncer.snapshot(ctx)}, nil
	})
}
