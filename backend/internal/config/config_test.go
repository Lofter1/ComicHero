package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFilesPreservesExistingValues(t *testing.T) {
	t.Setenv("CONFIG_EXISTING", "shell")
	t.Setenv("CONFIG_FROM_FILE", "")

	path := filepath.Join(t.TempDir(), ".env")
	contents := []byte("CONFIG_EXISTING=file\nexport CONFIG_FROM_FILE='loaded'\n# ignored\n")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := LoadEnvFiles(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("CONFIG_EXISTING"); got != "shell" {
		t.Fatalf("CONFIG_EXISTING = %q, want shell", got)
	}
	if got := os.Getenv("CONFIG_FROM_FILE"); got != "loaded" {
		t.Fatalf("CONFIG_FROM_FILE = %q, want loaded", got)
	}
}

func TestFromEnvUsesDefaultsAndExplicitlyDisablesAccessLog(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DB_PATH", "")
	t.Setenv("ACCESS_LOG_PATH", "")
	t.Setenv("METRON_BASE_URL", "")

	got := FromEnv("test-version")
	if got.Version != "test-version" || got.Address != ":8080" {
		t.Fatalf("unexpected identity config: %+v", got)
	}
	if got.DatabasePath != "./data/comicorder.db" {
		t.Fatalf("DatabasePath = %q", got.DatabasePath)
	}
	if got.AccessLogPath != "" {
		t.Fatalf("AccessLogPath = %q, want disabled", got.AccessLogPath)
	}
	if got.MetronBaseURL != "" {
		t.Fatalf("MetronBaseURL = %q, want Metron client default", got.MetronBaseURL)
	}
}
