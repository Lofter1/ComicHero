package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
)

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
