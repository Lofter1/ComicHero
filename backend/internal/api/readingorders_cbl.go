package api

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

const cblNoLocalComicMatch = "no local comic matched"

var cblPartPattern = regexp.MustCompile(`(?i)(?:^|[\s._-])(?:part|pt)\s*[._-]?\s*0*(\d+)(?:[\s._-]|$)`)

type cblReadingList struct {
	XMLName   xml.Name  `xml:"ReadingList"`
	XMLNSXSD  string    `xml:"xmlns:xsd,attr,omitempty"`
	XMLNSXSI  string    `xml:"xmlns:xsi,attr,omitempty"`
	Name      string    `xml:"Name"`
	NumIssues int       `xml:"NumIssues"`
	Books     []cblBook `xml:"Books>Book"`
	Matchers  string    `xml:"Matchers"`
}

type cblBook struct {
	Series    string        `xml:"Series,attr"`
	Number    string        `xml:"Number,attr"`
	Volume    string        `xml:"Volume,attr,omitempty"`
	Year      string        `xml:"Year,attr,omitempty"`
	Databases []cblDatabase `xml:"Database,omitempty"`
}

type cblDatabase struct {
	Name   string `xml:"Name,attr"`
	Series string `xml:"Series,attr,omitempty"`
	Issue  string `xml:"Issue,attr,omitempty"`
}

type cblImportDocument struct {
	name string
	list cblReadingList
	part int
}

type cblImportResult struct {
	readingOrder   ReadingOrderDetail
	matchedCount   int
	unmatchedCount int
	unmatched      []ReadingOrderCBLUnmatchedBook
}

type cblMissingComicResolver func(context.Context, cblBook) (*Comic, error)

func importReadingOrderCBL(ctx context.Context, db *sqlx.DB, input *ReadingOrderCBLImportInput) (*ReadingOrderCBLImportOutput, error) {
	documents, err := parseCBLImportDocuments(input)
	if err != nil {
		return nil, err
	}
	if len(documents) == 1 {
		result, err := importCBLDocument(ctx, db, documents[0])
		if err != nil {
			return nil, err
		}
		return cblImportOutput(result), nil
	}
	return importMultipartReadingOrderCBL(ctx, db, documents)
}

func parseCBLImportDocuments(input *ReadingOrderCBLImportInput) ([]cblImportDocument, error) {
	if input == nil {
		return nil, huma.Error400BadRequest("CBL content is required")
	}

	parts := input.Body.Parts
	if strings.TrimSpace(input.Body.Content) != "" {
		if len(parts) > 0 {
			return nil, huma.Error400BadRequest("provide either one CBL document or multipart CBL files, not both")
		}
		parts = []ReadingOrderCBLImportPart{{
			Filename: input.Body.Filename,
			Content:  input.Body.Content,
		}}
	}
	if len(parts) == 0 {
		return nil, huma.Error400BadRequest("CBL content is required")
	}
	if len(parts) > 100 {
		return nil, huma.Error400BadRequest("a multipart CBL import supports at most 100 files")
	}

	documents := make([]cblImportDocument, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part.Content) == "" {
			return nil, huma.Error400BadRequest("each CBL part must include content")
		}
		var cbl cblReadingList
		if err := xml.Unmarshal([]byte(part.Content), &cbl); err != nil {
			return nil, huma.Error400BadRequest("invalid CBL XML in " + cblImportPartLabel(part.Filename))
		}
		name := strings.TrimSpace(cbl.Name)
		if name == "" {
			name = readingOrderNameFromCBLFilename(part.Filename)
		}
		if name == "" {
			name = "Imported CBL reading order"
		}
		documents = append(documents, cblImportDocument{
			name: name,
			list: cbl,
		})
	}
	return documents, nil
}

func importMultipartReadingOrderCBL(ctx context.Context, db *sqlx.DB, documents []cblImportDocument) (*ReadingOrderCBLImportOutput, error) {
	name, err := prepareMultipartCBLDocuments(documents)
	if err != nil {
		return nil, err
	}
	result, err := importCombinedCBLDocuments(ctx, db, 0, name, documents)
	if err != nil {
		return nil, err
	}
	return cblImportOutput(result), nil
}

func importCombinedCBLDocuments(ctx context.Context, db *sqlx.DB, readingOrderID int, name string, documents []cblImportDocument) (cblImportResult, error) {
	return importCombinedCBLDocumentsWithResolver(ctx, db, readingOrderID, name, documents, nil)
}

func importCombinedCBLDocumentsWithResolver(ctx context.Context, db *sqlx.DB, readingOrderID int, name string, documents []cblImportDocument, resolveMissing cblMissingComicResolver) (cblImportResult, error) {
	entries := make([]ReadingOrderEntryPayload, 0)
	unmatched := make([]ReadingOrderCBLUnmatchedBook, 0)
	matchedCount := 0
	for _, document := range documents {
		entries = append(entries, ReadingOrderEntryPayload{Type: "section", Title: document.name})
		partEntries, partUnmatched, err := cblDocumentEntriesWithResolver(ctx, db, document.list, resolveMissing)
		if err != nil {
			return cblImportResult{}, err
		}
		entries = append(entries, partEntries...)
		matchedCount += len(partEntries)
		for _, book := range partUnmatched {
			book.Part = document.name
			unmatched = append(unmatched, book)
		}
	}

	createdReadingOrder := readingOrderID == 0
	if createdReadingOrder {
		created, err := createReadingOrder(ctx, db, nil, ReadingOrderPayload{Name: name})
		if err != nil {
			return cblImportResult{}, err
		}
		readingOrderID = created.Body.ID
	} else if _, err := db.ExecContext(ctx, `UPDATE reading_orders SET name = ? WHERE id = ?`, name, readingOrderID); err != nil {
		return cblImportResult{}, huma.Error500InternalServerError("failed to update multipart CBL reading order")
	}

	setInput := &SetReadingOrderComicsInput{ID: readingOrderID}
	setInput.Body.Entries = entries
	detail, err := setReadingOrderComicsInternal(ctx, db, setInput)
	if err != nil {
		if createdReadingOrder {
			cleanupCBLReadingOrders(ctx, db, []int{readingOrderID})
		}
		return cblImportResult{}, err
	}
	return cblImportResult{
		readingOrder:   detail.Body,
		matchedCount:   matchedCount,
		unmatchedCount: len(unmatched),
		unmatched:      unmatched,
	}, nil
}

func importCBLDocument(ctx context.Context, db *sqlx.DB, document cblImportDocument) (cblImportResult, error) {
	return importCBLDocumentWithResolver(ctx, db, document, nil)
}

func importCBLDocumentWithResolver(ctx context.Context, db *sqlx.DB, document cblImportDocument, resolveMissing cblMissingComicResolver) (cblImportResult, error) {
	entries, unmatched, err := cblDocumentEntriesWithResolver(ctx, db, document.list, resolveMissing)
	if err != nil {
		return cblImportResult{}, err
	}

	created, err := createReadingOrder(ctx, db, nil, ReadingOrderPayload{Name: document.name})
	if err != nil {
		return cblImportResult{}, err
	}

	setInput := &SetReadingOrderComicsInput{ID: created.Body.ID}
	setInput.Body.Entries = entries
	detail, err := setReadingOrderComics(ctx, db, setInput)
	if err != nil {
		return cblImportResult{}, err
	}

	return cblImportResult{
		readingOrder:   detail.Body,
		matchedCount:   len(entries),
		unmatchedCount: len(unmatched),
		unmatched:      unmatched,
	}, nil
}

func updateCBLDocument(ctx context.Context, db *sqlx.DB, readingOrderID int, document cblImportDocument) (cblImportResult, error) {
	return updateCBLDocumentWithResolver(ctx, db, readingOrderID, document, nil)
}

func updateCBLDocumentWithResolver(ctx context.Context, db *sqlx.DB, readingOrderID int, document cblImportDocument, resolveMissing cblMissingComicResolver) (cblImportResult, error) {
	entries, unmatched, err := cblDocumentEntriesWithResolver(ctx, db, document.list, resolveMissing)
	if err != nil {
		return cblImportResult{}, err
	}
	if _, err := db.ExecContext(ctx, `UPDATE reading_orders SET name = ? WHERE id = ?`, document.name, readingOrderID); err != nil {
		return cblImportResult{}, huma.Error500InternalServerError("failed to update CBL reading order")
	}
	setInput := &SetReadingOrderComicsInput{ID: readingOrderID}
	setInput.Body.Entries = entries
	detail, err := setReadingOrderComicsInternal(ctx, db, setInput)
	if err != nil {
		return cblImportResult{}, err
	}
	return cblImportResult{
		readingOrder:   detail.Body,
		matchedCount:   len(entries),
		unmatchedCount: len(unmatched),
		unmatched:      unmatched,
	}, nil
}

func cblDocumentEntries(ctx context.Context, db *sqlx.DB, cbl cblReadingList) ([]ReadingOrderEntryPayload, []ReadingOrderCBLUnmatchedBook, error) {
	return cblDocumentEntriesWithResolver(ctx, db, cbl, nil)
}

func cblDocumentEntriesWithResolver(ctx context.Context, db *sqlx.DB, cbl cblReadingList, resolveMissing cblMissingComicResolver) ([]ReadingOrderEntryPayload, []ReadingOrderCBLUnmatchedBook, error) {
	entries := make([]ReadingOrderEntryPayload, 0, len(cbl.Books))
	unmatched := make([]ReadingOrderCBLUnmatchedBook, 0)
	for i, book := range cbl.Books {
		comic, reason, err := matchCBLBook(ctx, db, book)
		if err != nil {
			return nil, nil, err
		}
		if comic == nil && reason == cblNoLocalComicMatch {
			if resolveMissing != nil {
				comic, err = resolveMissing(ctx, book)
				if err != nil {
					return nil, nil, err
				}
			}
		}
		if comic == nil && reason == cblNoLocalComicMatch {
			comic, err = createComicFromCBLBook(ctx, db, book)
			if err != nil {
				return nil, nil, err
			}
		}
		if comic == nil {
			unmatched = append(unmatched, cblUnmatchedBook(i+1, book, reason))
			continue
		}
		entries = append(entries, ReadingOrderEntryPayload{
			Type:    "comic",
			ComicID: comic.ID,
		})
	}
	return entries, unmatched, nil
}

func cblImportOutput(result cblImportResult) *ReadingOrderCBLImportOutput {
	return &ReadingOrderCBLImportOutput{Body: ReadingOrderCBLImportResult{
		ReadingOrder:   result.readingOrder,
		MatchedCount:   result.matchedCount,
		UnmatchedCount: result.unmatchedCount,
		Unmatched:      result.unmatched,
	}}
}

func prepareMultipartCBLDocuments(documents []cblImportDocument) (string, error) {
	if len(documents) < 2 {
		return "", huma.Error400BadRequest("multipart CBL import requires at least two files")
	}

	parentName := ""
	seenParts := map[int]bool{}
	for i := range documents {
		base, part, ok := cblMultipartPartName(documents[i].name)
		if !ok {
			return "", huma.Error400BadRequest("multipart CBL names must end in Part NN")
		}
		if parentName == "" {
			parentName = base
		} else if !strings.EqualFold(parentName, base) {
			return "", huma.Error400BadRequest("multipart CBL files must share the same reading-order name")
		}
		if seenParts[part] {
			return "", huma.Error400BadRequest(fmt.Sprintf("multipart CBL contains duplicate Part %d", part))
		}
		seenParts[part] = true
		documents[i].part = part
	}

	sort.SliceStable(documents, func(i, j int) bool {
		if documents[i].part != documents[j].part {
			return documents[i].part < documents[j].part
		}
		return strings.ToLower(documents[i].name) < strings.ToLower(documents[j].name)
	})
	return parentName, nil
}

func cblMultipartPartName(name string) (string, int, bool) {
	name = strings.TrimSpace(name)
	matches := cblPartPattern.FindStringSubmatchIndex(name)
	if len(matches) < 4 {
		return "", 0, false
	}
	part, ok := parseCBLPositiveInt(name[matches[2]:matches[3]])
	base := strings.Trim(strings.TrimSpace(name[:matches[0]]), "-_. ")
	return base, part, ok && base != ""
}

func cblImportPartLabel(filename string) string {
	if name := filepath.Base(strings.TrimSpace(filename)); name != "" && name != "." {
		return name
	}
	return "CBL part"
}

func cleanupCBLReadingOrders(ctx context.Context, db *sqlx.DB, ids []int) {
	for i := len(ids) - 1; i >= 0; i-- {
		_, _ = db.ExecContext(ctx, `DELETE FROM reading_order_comics WHERE reading_order_id = ?`, ids[i])
		_, _ = db.ExecContext(ctx, `DELETE FROM reading_order_children WHERE parent_reading_order_id = ? OR child_reading_order_id = ?`, ids[i], ids[i])
		_, _ = db.ExecContext(ctx, `DELETE FROM reading_order_sections WHERE reading_order_id = ?`, ids[i])
		_, _ = db.ExecContext(ctx, `DELETE FROM reading_orders WHERE id = ?`, ids[i])
	}
}

func exportReadingOrderCBL(ctx context.Context, db *sqlx.DB, id int) (*ReadingOrderCBLExportOutput, error) {
	detail, err := getReadingOrder(ctx, db, id)
	if err != nil {
		return nil, err
	}

	books := make([]cblBook, 0, len(detail.Body.Comics))
	for _, comic := range detail.Body.Comics {
		book := cblBook{
			Series: strings.TrimSpace(comic.Series),
			Number: strings.TrimSpace(comic.Issue),
		}
		if comic.SeriesYear > 0 {
			book.Volume = strconv.Itoa(comic.SeriesYear)
		}
		if year := comicCBLYear(comic); year != "" {
			book.Year = year
		}
		if comic.ComicVineID != nil && *comic.ComicVineID > 0 {
			book.Databases = append(book.Databases, cblDatabase{Name: "comicvine", Issue: strconv.Itoa(*comic.ComicVineID)})
		}
		if comic.MetronIssueID != nil && *comic.MetronIssueID > 0 {
			book.Databases = append(book.Databases, cblDatabase{Name: "metron", Issue: strconv.Itoa(*comic.MetronIssueID)})
		}
		books = append(books, book)
	}

	cbl := cblReadingList{
		XMLNSXSD:  "http://www.w3.org/2001/XMLSchema",
		XMLNSXSI:  "http://www.w3.org/2001/XMLSchema-instance",
		Name:      detail.Body.Name,
		NumIssues: len(books),
		Books:     books,
	}
	out, err := xml.MarshalIndent(cbl, "", "  ")
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to build CBL")
	}

	return &ReadingOrderCBLExportOutput{Body: ReadingOrderCBLExport{
		Filename: cblFilename(detail.Body.Name),
		Content:  xml.Header + string(out) + "\n",
	}}, nil
}

func matchCBLBook(ctx context.Context, db *sqlx.DB, book cblBook) (*Comic, string, error) {
	comicVineID, hasComicVineID := cblComicVineID(book)
	if hasComicVineID {
		comic, found, err := fetchCBLComicByComicVineID(ctx, db, comicVineID)
		if err != nil {
			return nil, "", err
		}
		if found {
			return comic, "", nil
		}
	}

	series := strings.TrimSpace(book.Series)
	number := strings.TrimSpace(book.Number)
	if series == "" || number == "" {
		return nil, "missing series or issue number", nil
	}

	volume, hasVolume := parseCBLPositiveInt(book.Volume)
	year, hasYear := parseCBLPositiveInt(book.Year)

	var comic *Comic
	var reason string
	var err error
	switch {
	case hasVolume:
		comic, reason, err = fetchCBLComicMatch(ctx, db, `
			SELECT * FROM comics
			WHERE LOWER(series) = LOWER(?) AND LOWER(issue) = LOWER(?) AND series_year = ?
			ORDER BY id
			LIMIT 2
		`, series, number, volume)
	case hasYear:
		comic, reason, err = fetchCBLComicMatch(ctx, db, `
			SELECT * FROM comics
			WHERE LOWER(series) = LOWER(?) AND LOWER(issue) = LOWER(?)
				AND (series_year = ? OR substr(cover_date, 1, 4) = ?)
			ORDER BY id
			LIMIT 2
		`, series, number, year, strconv.Itoa(year))
	default:
		comic, reason, err = fetchCBLComicMatch(ctx, db, `
			SELECT * FROM comics
			WHERE LOWER(series) = LOWER(?) AND LOWER(issue) = LOWER(?)
			ORDER BY id
			LIMIT 2
		`, series, number)
	}
	if err != nil || comic == nil || !hasComicVineID {
		return comic, reason, err
	}
	if err := attachComicVineID(ctx, db, comic.ID, comicVineID); err != nil {
		return nil, "", err
	}
	comic.ComicVineID = &comicVineID
	return comic, "", nil
}

func fetchCBLComicByComicVineID(ctx context.Context, db *sqlx.DB, comicVineID int) (*Comic, bool, error) {
	var comic Comic
	if err := db.GetContext(ctx, &comic, `SELECT * FROM comics WHERE comic_vine_id = ?`, comicVineID); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, huma.Error500InternalServerError("failed to match CBL Comic Vine ID")
	}
	hydrateComicTitle(&comic)
	return &comic, true, nil
}

func fetchCBLComicMatch(ctx context.Context, db *sqlx.DB, query string, args ...any) (*Comic, string, error) {
	matches := []Comic{}
	if err := db.SelectContext(ctx, &matches, query, args...); err != nil {
		return nil, "", huma.Error500InternalServerError("failed to match CBL book")
	}
	if len(matches) == 0 {
		return nil, cblNoLocalComicMatch, nil
	}
	if len(matches) > 1 {
		return nil, "multiple local comics matched", nil
	}
	hydrateComicTitle(&matches[0])
	return &matches[0], "", nil
}

func createComicFromCBLBook(ctx context.Context, db *sqlx.DB, book cblBook) (*Comic, error) {
	series := strings.TrimSpace(book.Series)
	number := strings.TrimSpace(book.Number)
	if series == "" || number == "" {
		return nil, nil
	}

	seriesYear, _ := parseCBLPositiveInt(book.Volume)
	if seriesYear == 0 {
		seriesYear, _ = parseCBLPositiveInt(book.Year)
	}
	seriesID, err := ensureSeriesRow(ctx, db, series, seriesYear)
	if err != nil {
		return nil, err
	}
	comicVineID, _ := cblComicVineID(book)
	result, err := db.ExecContext(ctx, `
		INSERT INTO comics (series_id, series, series_year, issue, publisher, cover_date, cover_image, description, comic_vine_id)
		VALUES (?, ?, ?, ?, '', '', '', '', ?)
	`, nullableSeriesID(seriesID), series, seriesYear, number, nullablePositiveID(comicVineID))
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create comic from CBL")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get CBL comic id")
	}
	comic := Comic{
		ID:          int(id),
		SeriesID:    intPointerOrNil(seriesID),
		Series:      series,
		SeriesYear:  seriesYear,
		Issue:       number,
		ComicVineID: intPointerOrNil(comicVineID),
	}
	hydrateComicTitle(&comic)
	return &comic, nil
}

func cblComicVineID(book cblBook) (int, bool) {
	for _, database := range book.Databases {
		name := strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return unicode.ToLower(r)
			}
			return -1
		}, database.Name)
		if name != "cv" && name != "cvdb" && name != "comicvine" && name != "comicvinedb" && name != "comicvinedatabase" {
			continue
		}
		parts := strings.FieldsFunc(database.Issue, func(r rune) bool { return !unicode.IsDigit(r) })
		for i := len(parts) - 1; i >= 0; i-- {
			if id, ok := parseCBLPositiveInt(parts[i]); ok {
				return id, true
			}
		}
	}
	return 0, false
}

func intPointerOrNil(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}

func cblUnmatchedBook(position int, book cblBook, reason string) ReadingOrderCBLUnmatchedBook {
	if reason == "" {
		reason = cblNoLocalComicMatch
	}
	return ReadingOrderCBLUnmatchedBook{
		Position: position,
		Series:   strings.TrimSpace(book.Series),
		Number:   strings.TrimSpace(book.Number),
		Volume:   strings.TrimSpace(book.Volume),
		Year:     strings.TrimSpace(book.Year),
		Reason:   reason,
	}
}

func readingOrderNameFromCBLFilename(filename string) string {
	base := filepath.Base(strings.TrimSpace(filename))
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	ext := filepath.Ext(base)
	if strings.EqualFold(ext, ".cbl") {
		base = strings.TrimSuffix(base, ext)
	}
	return strings.TrimSpace(base)
}

func parseCBLPositiveInt(value string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	return n, err == nil && n > 0
}

func comicCBLYear(comic ReadingOrderComic) string {
	if len(comic.CoverDate) >= 4 {
		year := comic.CoverDate[:4]
		if _, ok := parseCBLPositiveInt(year); ok {
			return year
		}
	}
	if comic.SeriesYear > 0 {
		return strconv.Itoa(comic.SeriesYear)
	}
	return ""
}

func cblFilename(name string) string {
	stem := strings.TrimSpace(name)
	if stem == "" {
		stem = "reading-order"
	}
	var out strings.Builder
	lastDash := false
	for _, r := range stem {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteRune('-')
			lastDash = true
		}
	}
	filename := strings.Trim(out.String(), "-")
	if filename == "" {
		filename = "reading-order"
	}
	return fmt.Sprintf("%s.cbl", filename)
}
