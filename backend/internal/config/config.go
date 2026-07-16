// Package config owns process-level configuration and .env loading.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Config contains the runtime settings needed to assemble the application.
type Config struct {
	Version         string
	Address         string
	DatabasePath    string
	AccessLogPath   string
	CoverCacheDir   string
	CoverPublicPath string
	StaticDir       string
	MetronBaseURL   string
	MetronUsername  string
	MetronPassword  string
}

// FromEnv reads runtime configuration after any .env files have been loaded.
func FromEnv(version string) Config {
	accessLogPath := "./data/access.log"
	if configuredPath, configured := os.LookupEnv("ACCESS_LOG_PATH"); configured {
		accessLogPath = configuredPath
	}

	return Config{
		Version:         version,
		Address:         ":" + value("PORT", "8080"),
		DatabasePath:    value("DB_PATH", "./data/comicorder.db"),
		AccessLogPath:   accessLogPath,
		CoverCacheDir:   value("COVER_CACHE_DIR", "./public/covers"),
		CoverPublicPath: "/covers",
		StaticDir:       os.Getenv("STATIC_DIR"),
		MetronBaseURL:   os.Getenv("METRON_BASE_URL"),
		MetronUsername:  os.Getenv("METRON_USERNAME"),
		MetronPassword:  os.Getenv("METRON_PASSWORD"),
	}
}

// LoadEnvFiles loads the first value found for each variable from the supplied
// files. Missing files are intentionally ignored so local overrides remain
// optional.
func LoadEnvFiles(paths ...string) error {
	var loadErrors []error
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				loadErrors = append(loadErrors, fmt.Errorf("open %q: %w", path, err))
			}
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			setEnvLine(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("read %q: %w", path, err))
		}
		if err := file.Close(); err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("close %q: %w", path, err))
		}
	}
	return errors.Join(loadErrors...)
}

func value(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func setEnvLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return
	}
	line = strings.TrimPrefix(line, "export ")
	key, value, ok := strings.Cut(line, "=")
	if !ok {
		return
	}

	key = strings.TrimSpace(key)
	if key == "" || os.Getenv(key) != "" {
		return
	}

	value = strings.Trim(strings.TrimSpace(value), `"'`)
	_ = os.Setenv(key, value)
}
