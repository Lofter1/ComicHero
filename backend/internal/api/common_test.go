package api

import (
	"context"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestParseOptionalBool(t *testing.T) {
	value, ok, err := parseOptionalBool("", "favorite")
	if err != nil || ok || value {
		t.Fatalf("empty value = %v, %v, %v; want false, false, nil", value, ok, err)
	}

	value, ok, err = parseOptionalBool("true", "favorite")
	if err != nil || !ok || !value {
		t.Fatalf("true value = %v, %v, %v; want true, true, nil", value, ok, err)
	}

	if _, _, err := parseOptionalBool("sometimes", "favorite"); err == nil {
		t.Fatal("invalid bool returned nil error")
	}
}

func TestPaginationHelpers(t *testing.T) {
	query, args, limit, offset := paginatedQuery("SELECT * FROM comics", []any{"arg"}, 250, -12)
	if query != "SELECT * FROM comics LIMIT ? OFFSET ?" {
		t.Fatalf("query = %q", query)
	}
	if limit != maxPageLimit || offset != 0 {
		t.Fatalf("limit/offset = %d/%d; want %d/0", limit, offset, maxPageLimit)
	}
	if len(args) != 3 || args[1] != maxPageLimit+1 || args[2] != 0 {
		t.Fatalf("args = %#v; want original arg plus page limit+1 and offset", args)
	}

	items, headers := pageItems([]int{1, 2, 3}, 2, 10, 42)
	if len(items) != 2 || items[0] != 1 || items[1] != 2 {
		t.Fatalf("items = %#v; want first two", items)
	}
	if headers.PageLimit != "2" || headers.PageOffset != "10" || headers.HasMore != "true" || headers.TotalCount != "42" {
		t.Fatalf("headers = %#v; want limit 2, offset 10, has more true, total 42", headers)
	}
}

func testUserContext() context.Context {
	return context.WithValue(context.Background(), contextUserIDKey{}, 1)
}

func TestComicListQuery(t *testing.T) {
	query, args, err := comicListQuery(&ComicListInput{
		Query:          "bat",
		Series:         "Detective",
		Publisher:      "DC",
		Read:           "false",
		ReadingOrderID: 12,
	}, 1)
	if err != nil {
		t.Fatalf("comicListQuery returned error: %v", err)
	}

	for _, fragment := range []string{
		"c.series LIKE ?",
		"c.series_year AS TEXT",
		"c.issue AS TEXT",
		"c.publisher LIKE ?",
		"COALESCE(uc.read, 0) = ?",
		"roc.reading_order_id = ?",
		"ORDER BY c.series, c.series_year, CAST(c.issue AS REAL), c.issue",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("query missing %q: %s", fragment, query)
		}
	}
	if len(args) != 11 {
		t.Fatalf("len(args) = %d; want 11", len(args))
	}

	query, _, err = comicListQuery(&ComicListInput{Status: "read,skipped"}, 1)
	if err != nil {
		t.Fatalf("comicListQuery status returned error: %v", err)
	}
	for _, fragment := range []string{
		"COALESCE(uc.read, 0) = 1",
		"COALESCE(uc.skipped, 0) = 1",
		" OR ",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("status query missing %q: %s", fragment, query)
		}
	}
}

func TestComicListQueryNormalizesTitleStyleYear(t *testing.T) {
	_, args, err := comicListQuery(&ComicListInput{Query: "Ultimates (2016) #4"}, 1)
	if err != nil {
		t.Fatalf("comicListQuery returned error: %v", err)
	}

	if len(args) != 12 {
		t.Fatalf("args = %#v; want user, issue, and two five-column search terms", args)
	}
	if args[1] != "4" {
		t.Fatalf("issue arg = %#v; want 4", args[1])
	}
	for index, arg := range args[7:] {
		if arg != "%2016%" {
			t.Fatalf("year arg %d = %#v; want %%2016%%", index, arg)
		}
	}
}
