package app

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type accessLogger struct {
	file *os.File
	mu   sync.Mutex
}

type accessLogEntry struct {
	Timestamp    string `json:"timestamp"`
	Method       string `json:"method"`
	Path         string `json:"path"`
	Status       int    `json:"status"`
	DurationMS   int64  `json:"durationMs"`
	Bytes        int    `json:"bytes"`
	RemoteAddr   string `json:"remoteAddr"`
	ForwardedFor string `json:"forwardedFor,omitempty"`
	UserAgent    string `json:"userAgent,omitempty"`
}

func newAccessLogger(path string) (*accessLogger, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return &accessLogger{}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, err
	}
	return &accessLogger{file: file}, nil
}

func (l *accessLogger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func (l *accessLogger) Middleware(next http.Handler) http.Handler {
	if l == nil || l.file == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(wrapped, r)
		status := wrapped.Status()
		if status == 0 {
			status = http.StatusOK
		}
		entry := accessLogEntry{
			Timestamp:    started.UTC().Format(time.RFC3339Nano),
			Method:       r.Method,
			Path:         r.URL.Path,
			Status:       status,
			DurationMS:   time.Since(started).Milliseconds(),
			Bytes:        wrapped.BytesWritten(),
			RemoteAddr:   r.RemoteAddr,
			ForwardedFor: r.Header.Get("X-Forwarded-For"),
			UserAgent:    r.UserAgent(),
		}
		encoded, err := json.Marshal(entry)
		if err != nil {
			return
		}
		l.mu.Lock()
		_, _ = l.file.Write(append(encoded, '\n'))
		l.mu.Unlock()
	})
}
