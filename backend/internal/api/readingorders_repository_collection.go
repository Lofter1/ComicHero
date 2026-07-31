package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// cblCollectionSequencePattern matches a standalone 1-3 digit number
// anywhere in a CBL filename, as used by the DieselTech/CBL-ReadingLists
// naming conventions for a collection's sequence position — e.g. "01 -
// Silver Age Spider-Man.cbl" or "Spider-Man 001 (Silver Age).cbl". The
// word-boundary anchors deliberately exclude 4-digit numbers so a cover
// year like "2099" in a title isn't mistaken for a sequence number. This is
// a best-effort heuristic, not a guarantee — a title that happens to
// contain an unrelated 1-3 digit number (e.g. an issue number in the name)
// can occasionally produce a misleading sequence. It's used only for
// display/ordering, not for merging or any data-integrity-sensitive logic,
// so a wrong guess here has low impact. This is intentionally independent
// from cblPartPattern ("Part N" suffix, used to *merge* multiple files into
// one reading order) — a numbered collection file is its own distinct,
// separately-readable reading order, just organized under a shared folder,
// and should not be merged with its siblings.
var cblCollectionSequencePattern = regexp.MustCompile(`\b0*(\d{1,3})\b`)

// deriveCollectionMetadata derives a human-readable collection name (the
// file's immediate parent folder) and, when the filename follows the
// DieselTech numeric-prefix convention, its sequence position within that
// collection. It returns an empty collection ("", nil) for files at the
// repository root, which have no meaningful grouping folder.
func deriveCollectionMetadata(filePath string) (collection string, sequence *int) {
	dir := path.Dir(filePath)
	if dir == "." || dir == "/" || dir == "" {
		return "", nil
	}
	collection = path.Base(dir)

	base := strings.TrimSuffix(path.Base(filePath), path.Ext(filePath))
	if matches := cblCollectionSequencePattern.FindStringSubmatch(base); len(matches) == 2 {
		if n, err := strconv.Atoi(matches[1]); err == nil {
			sequence = &n
		}
	}
	return collection, sequence
}

// fetchFolderReadme fetches the raw content of a README.md sitting alongside
// a CBL file's containing folder, for use as a reading order's description
// when one isn't already set. It returns ("", nil) rather than an error when
// no README is present, since most folders in a CBL repository won't have
// one — that's an expected, unremarkable case, not a failure.
func (s *cblRepositorySyncer) fetchFolderReadme(
	ctx context.Context,
	repository cblGitHubRepository,
	branch, folderPath string,
	cache map[string]string,
) (string, error) {
	if folderPath == "." || folderPath == "" {
		return "", nil
	}
	if content, ok := cache[folderPath]; ok {
		return content, nil
	}

	readmePath := path.Join(folderPath, "README.md")
	target, err := url.Parse(s.rawBase)
	if err != nil {
		return "", err
	}
	target.Path = path.Join(target.Path, repository.Owner, repository.Name, branch, readmePath)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "ComicHero-CBL-Importer")

	response, err := s.httpClient.Do(request)
	if err != nil {
		// Treat a network hiccup fetching an optional README the same as
		// "not present" rather than failing the whole sync over it.
		cache[folderPath] = ""
		return "", nil
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode == http.StatusNotFound {
		cache[folderPath] = ""
		return "", nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("download %s: GitHub returned %s", readmePath, response.Status)
	}

	content, err := io.ReadAll(io.LimitReader(response.Body, cblRepositoryMaxFileSize))
	if err != nil {
		return "", err
	}

	text := strings.TrimSpace(string(content))
	cache[folderPath] = text
	return text, nil
}

// backfillReadingOrderDescription sets a reading order's description from
// the given text only when it doesn't already have one, so a user's own
// edits are never overwritten by a sync run.
func (s *cblRepositorySyncer) backfillReadingOrderDescription(ctx context.Context, readingOrderID int, description string) error {
	description = strings.TrimSpace(description)
	if description == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE reading_orders
		SET description = ?
		WHERE id = ? AND TRIM(description) = ''
	`, description, readingOrderID)
	return err
}
