package app

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAccessLoggerPersistsRequestsWithoutQueryStrings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "access.log")
	logger, err := newAccessLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	handler := logger.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/reset-password?token=secret", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.2")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if err := logger.Close(); err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	var entry accessLogEntry
	if !bufio.NewScanner(file).Scan() {
		t.Fatal("access log is empty")
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := json.NewDecoder(file).Decode(&entry); err != nil {
		t.Fatal(err)
	}
	if entry.Path != "/api/auth/reset-password" || entry.Status != http.StatusUnauthorized || entry.RemoteAddr != "192.0.2.10:1234" || entry.ForwardedFor != "198.51.100.2" {
		t.Fatalf("access log entry = %+v", entry)
	}
}

func TestAccessLoggerCanBeDisabled(t *testing.T) {
	logger, err := newAccessLogger("")
	if err != nil {
		t.Fatal(err)
	}
	if logger.file != nil {
		t.Fatal("empty access log path should disable file logging")
	}
}
