package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type CoverCache struct {
	dir        string
	publicPath string
	client     *http.Client
}

func NewCoverCache(dir, publicPath string) *CoverCache {
	return &CoverCache{
		dir:        dir,
		publicPath: "/" + strings.Trim(strings.TrimSpace(publicPath), "/"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *CoverCache) EnsureDir() error {
	if c == nil {
		return nil
	}
	return os.MkdirAll(c.dir, 0o755)
}

func (c *CoverCache) LocalURL(ctx context.Context, source string) (string, error) {
	source = strings.TrimSpace(source)
	if source == "" || c == nil {
		return source, nil
	}

	parsed, err := url.Parse(source)
	if err != nil || !parsed.IsAbs() {
		return source, nil
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return source, nil
	}

	name := coverFileName(source, parsed)
	localPath := filepath.Join(c.dir, name)
	if _, err := os.Stat(localPath); err == nil {
		return c.coverURL(name), nil
	}

	if err := c.EnsureDir(); err != nil {
		return "", fmt.Errorf("prepare cover cache: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return "", fmt.Errorf("create cover request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download cover: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("download cover: %s", resp.Status)
	}

	if ext := coverExtensionFromContentType(resp.Header.Get("Content-Type")); ext != "" && filepath.Ext(name) == "" {
		name += ext
		localPath = filepath.Join(c.dir, name)
		if _, err := os.Stat(localPath); err == nil {
			return c.coverURL(name), nil
		}
	}

	tmp, err := os.CreateTemp(c.dir, "cover-*")
	if err != nil {
		return "", fmt.Errorf("create cover file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return "", fmt.Errorf("save cover: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close cover file: %w", err)
	}
	if err := os.Rename(tmpPath, localPath); err != nil {
		return "", fmt.Errorf("store cover: %w", err)
	}

	return c.coverURL(name), nil
}

func (c *CoverCache) coverURL(name string) string {
	return path.Join(c.publicPath, name)
}

func coverFileName(source string, parsed *url.URL) string {
	sum := sha256.Sum256([]byte(source))
	hash := hex.EncodeToString(sum[:])[:24]
	ext := strings.ToLower(path.Ext(parsed.Path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif":
		return hash + ext
	default:
		return hash
	}
}

func coverExtensionFromContentType(contentType string) string {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	switch mediaType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/avif":
		return ".avif"
	default:
		return ""
	}
}
