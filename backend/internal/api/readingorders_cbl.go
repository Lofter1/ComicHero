package api

import (
	"context"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

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

func importReadingOrderCBL(ctx context.Context, db *sqlx.DB, input *ReadingOrderCBLImportInput) (*ReadingOrderCBLImportOutput, error) {
	var cbl cblReadingList
	if err := xml.Unmarshal([]byte(input.Body.Content), &cbl); err != nil {
		return nil, huma.Error400BadRequest("invalid CBL XML")
	}

	name := strings.TrimSpace(cbl.Name)
	if name == "" {
		name = readingOrderNameFromCBLFilename(input.Body.Filename)
	}
	if name == "" {
		name = "Imported CBL reading order"
	}

	entries := make([]ReadingOrderEntryPayload, 0, len(cbl.Books))
	unmatched := make([]ReadingOrderCBLUnmatchedBook, 0)
	for i, book := range cbl.Books {
		comic, reason, err := matchCBLBook(ctx, db, book)
		if err != nil {
			return nil, err
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

	created, err := createReadingOrder(ctx, db, ReadingOrderPayload{Name: name})
	if err != nil {
		return nil, err
	}

	setInput := &SetReadingOrderComicsInput{ID: created.Body.ID}
	setInput.Body.Entries = entries
	detail, err := setReadingOrderComics(ctx, db, setInput)
	if err != nil {
		return nil, err
	}

	return &ReadingOrderCBLImportOutput{Body: ReadingOrderCBLImportResult{
		ReadingOrder:   detail.Body,
		MatchedCount:   len(entries),
		UnmatchedCount: len(unmatched),
		Unmatched:      unmatched,
	}}, nil
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
		if comic.MetronIssueID != nil && *comic.MetronIssueID > 0 {
			book.Databases = []cblDatabase{{Name: "metron", Issue: strconv.Itoa(*comic.MetronIssueID)}}
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
	series := strings.TrimSpace(book.Series)
	number := strings.TrimSpace(book.Number)
	if series == "" || number == "" {
		return nil, "missing series or issue number", nil
	}

	volume, hasVolume := parseCBLPositiveInt(book.Volume)
	year, hasYear := parseCBLPositiveInt(book.Year)

	switch {
	case hasVolume:
		return fetchCBLComicMatch(ctx, db, `
			SELECT * FROM comics
			WHERE LOWER(series) = LOWER(?) AND LOWER(issue) = LOWER(?) AND series_year = ?
			ORDER BY id
			LIMIT 2
		`, series, number, volume)
	case hasYear:
		return fetchCBLComicMatch(ctx, db, `
			SELECT * FROM comics
			WHERE LOWER(series) = LOWER(?) AND LOWER(issue) = LOWER(?)
				AND (series_year = ? OR substr(cover_date, 1, 4) = ?)
			ORDER BY id
			LIMIT 2
		`, series, number, year, strconv.Itoa(year))
	default:
		return fetchCBLComicMatch(ctx, db, `
			SELECT * FROM comics
			WHERE LOWER(series) = LOWER(?) AND LOWER(issue) = LOWER(?)
			ORDER BY id
			LIMIT 2
		`, series, number)
	}
}

func fetchCBLComicMatch(ctx context.Context, db *sqlx.DB, query string, args ...any) (*Comic, string, error) {
	matches := []Comic{}
	if err := db.SelectContext(ctx, &matches, query, args...); err != nil {
		return nil, "", huma.Error500InternalServerError("failed to match CBL book")
	}
	if len(matches) == 0 {
		return nil, "no local comic matched", nil
	}
	if len(matches) > 1 {
		return nil, "multiple local comics matched", nil
	}
	hydrateComicTitle(&matches[0])
	return &matches[0], "", nil
}

func cblUnmatchedBook(position int, book cblBook, reason string) ReadingOrderCBLUnmatchedBook {
	if reason == "" {
		reason = "no local comic matched"
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
