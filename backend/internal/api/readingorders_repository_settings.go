package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

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
