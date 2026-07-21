package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestAuditMiddlewareRecordsOnlySuccessfulMutations(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`CREATE TABLE audit_events (id INTEGER PRIMARY KEY, user_id INTEGER, method TEXT, path TEXT, status_code INTEGER, occurred_at TEXT DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}

	handler := AuditMiddleware(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/failed" {
			http.Error(w, "failed", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, request := range []struct {
		method string
		path   string
	}{{http.MethodGet, "/comics"}, {http.MethodPatch, "/failed"}, {http.MethodPost, "/readingOrders"}} {
		req := httptest.NewRequest(request.method, request.path, nil)
		req = req.WithContext(context.WithValue(req.Context(), contextUserIDKey{}, 7))
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}

	var events []AuditEvent
	if err := db.Select(&events, `SELECT id, user_id, method, path, status_code, occurred_at, '' AS user_name FROM audit_events`); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].UserID == nil || *events[0].UserID != 7 || events[0].Method != http.MethodPost || events[0].Path != "/readingOrders" {
		t.Fatalf("audit events = %#v; want one successful POST by user 7", events)
	}
}

func TestListAuditEventsSearchesFiltersSortsAndPaginates(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE audit_events (
			id INTEGER PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
			method TEXT NOT NULL,
			path TEXT NOT NULL,
			status_code INTEGER NOT NULL,
			occurred_at TEXT NOT NULL
		);
		INSERT INTO users (id, name, is_admin) VALUES (2, 'Alice', 0), (3, 'Bob', 0);
		INSERT INTO audit_events (id, user_id, method, path, status_code, occurred_at) VALUES
			(1, 2, 'POST', '/comics', 201, '2026-07-20T10:00:00Z'),
			(2, 3, 'DELETE', '/users/4', 422, '2026-07-20T12:00:00Z'),
			(3, NULL, 'PATCH', '/settings', 302, '2026-07-20T11:00:00Z'),
			(4, 2, 'PUT', '/readingOrders/1', 200, '2026-07-20T13:00:00Z');
	`); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), contextUserIDKey{}, 1)

	t.Run("search", func(t *testing.T) {
		for _, test := range []struct {
			query string
			want  []int
		}{{"settings", []int{3}}, {"alice", []int{4, 1}}, {"201", []int{1}}} {
			result, err := listAuditEvents(ctx, db, &AuditEventListInput{Query: test.query})
			if err != nil {
				t.Fatal(err)
			}
			if got := auditEventIDs(result.Body); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("search %q IDs = %v, want %v", test.query, got, test.want)
			}
		}
	})

	t.Run("combined filters", func(t *testing.T) {
		result, err := listAuditEvents(ctx, db, &AuditEventListInput{
			UserID: 2,
			Method: "put",
			Status: "2xx",
		})
		if err != nil {
			t.Fatal(err)
		}
		if got := auditEventIDs(result.Body); !reflect.DeepEqual(got, []int{4}) {
			t.Fatalf("filtered IDs = %v, want [4]", got)
		}
	})

	t.Run("system events", func(t *testing.T) {
		result, err := listAuditEvents(ctx, db, &AuditEventListInput{System: true})
		if err != nil {
			t.Fatal(err)
		}
		if got := auditEventIDs(result.Body); !reflect.DeepEqual(got, []int{3}) {
			t.Fatalf("system IDs = %v, want [3]", got)
		}
	})

	t.Run("error status family", func(t *testing.T) {
		result, err := listAuditEvents(ctx, db, &AuditEventListInput{Status: "4xx"})
		if err != nil {
			t.Fatal(err)
		}
		if got := auditEventIDs(result.Body); !reflect.DeepEqual(got, []int{2}) {
			t.Fatalf("4xx IDs = %v, want [2]", got)
		}
	})

	t.Run("sort and pagination", func(t *testing.T) {
		result, err := listAuditEvents(ctx, db, &AuditEventListInput{
			Sort: "occurredAt", Direction: "desc", Limit: 2, Offset: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		if got := auditEventIDs(result.Body); !reflect.DeepEqual(got, []int{2, 3}) {
			t.Fatalf("paged IDs = %v, want [2 3]", got)
		}
		if result.TotalCount != "4" || result.PageLimit != "2" || result.PageOffset != "1" || result.HasMore != "true" {
			t.Fatalf("pagination = %#v", result.PaginationHeaders)
		}
	})

	t.Run("action sort", func(t *testing.T) {
		result, err := listAuditEvents(ctx, db, &AuditEventListInput{
			Sort: "action", Direction: "asc",
		})
		if err != nil {
			t.Fatal(err)
		}
		if got := auditEventIDs(result.Body); !reflect.DeepEqual(got, []int{2, 3, 1, 4}) {
			t.Fatalf("action-sorted IDs = %v, want [2 3 1 4]", got)
		}
	})
}

func auditEventIDs(events []AuditEvent) []int {
	ids := make([]int, len(events))
	for i, event := range events {
		ids[i] = event.ID
	}
	return ids
}
