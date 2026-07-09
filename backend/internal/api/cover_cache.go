package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	coverMaxDimension     = 720
	coverJPEGQuality      = 78
	coverDownloadMaxBytes = 20 << 20
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

func (c *CoverCache) Dir() string {
	if c == nil {
		return ""
	}
	return c.dir
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

	name := coverFileName(source)
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

	imageBytes, err := io.ReadAll(io.LimitReader(resp.Body, coverDownloadMaxBytes+1))
	if err != nil {
		return "", fmt.Errorf("read cover: %w", err)
	}
	if len(imageBytes) > coverDownloadMaxBytes {
		return "", fmt.Errorf("download cover: image is larger than %d bytes", coverDownloadMaxBytes)
	}

	optimized, err := optimizeCover(imageBytes)
	if err != nil {
		log.Printf("cover cache: keeping remote image %q because it could not be optimized: %v", source, err)
		return source, nil
	}

	tmp, err := os.CreateTemp(c.dir, "cover-*")
	if err != nil {
		return "", fmt.Errorf("create cover file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(optimized); err != nil {
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

func (c *CoverCache) StoreImage(source []byte) (string, error) {
	if c == nil {
		return "", fmt.Errorf("cover cache is not configured")
	}
	if len(source) == 0 {
		return "", fmt.Errorf("cover image is empty")
	}
	if len(source) > coverDownloadMaxBytes {
		return "", fmt.Errorf("cover image is larger than %d bytes", coverDownloadMaxBytes)
	}

	optimized, err := optimizeCover(source)
	if err != nil {
		return "", fmt.Errorf("optimize cover: %w", err)
	}

	name := coverBytesFileName(optimized)
	localPath := filepath.Join(c.dir, name)
	if _, err := os.Stat(localPath); err == nil {
		return c.coverURL(name), nil
	}

	if err := c.EnsureDir(); err != nil {
		return "", fmt.Errorf("prepare cover cache: %w", err)
	}

	tmp, err := os.CreateTemp(c.dir, "cover-*")
	if err != nil {
		return "", fmt.Errorf("create cover file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(optimized); err != nil {
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

func (c *CoverCache) RemoveLocalURL(source string) error {
	if c == nil {
		return nil
	}
	name, ok := c.localFileName(source)
	if !ok {
		return nil
	}
	if err := os.Remove(filepath.Join(c.dir, name)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (c *CoverCache) coverURL(name string) string {
	return path.Join(c.publicPath, name)
}

func (c *CoverCache) localFileName(source string) (string, bool) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", false
	}
	if parsed, err := url.Parse(source); err == nil && parsed.Path != "" {
		source = parsed.Path
	}

	publicPath := "/" + strings.Trim(strings.TrimSpace(c.publicPath), "/")
	if !strings.HasPrefix(source, publicPath+"/") {
		return "", false
	}
	name := path.Base(source)
	if name == "." || name == "/" || name == "" {
		return "", false
	}
	return name, true
}

func coverFileName(source string) string {
	sum := sha256.Sum256([]byte(source))
	hash := hex.EncodeToString(sum[:])[:24]
	return hash + ".jpg"
}

func coverBytesFileName(source []byte) string {
	sum := sha256.Sum256(source)
	hash := hex.EncodeToString(sum[:])[:24]
	return hash + ".jpg"
}

func optimizeCover(source []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(source))
	if err != nil {
		return nil, err
	}

	resized := flattenImage(resizeImage(img, coverMaxDimension))
	var out bytes.Buffer
	if err := jpeg.Encode(&out, resized, &jpeg.Options{Quality: coverJPEGQuality}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func resizeImage(src image.Image, maxDimension int) image.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 || maxDimension <= 0 {
		return src
	}
	if width <= maxDimension && height <= maxDimension {
		return src
	}

	scale := float64(maxDimension) / float64(width)
	if height > width {
		scale = float64(maxDimension) / float64(height)
	}
	dstWidth := max(1, int(float64(width)*scale))
	dstHeight := max(1, int(float64(height)*scale))
	dst := image.NewRGBA(image.Rect(0, 0, dstWidth, dstHeight))
	for y := 0; y < dstHeight; y++ {
		srcY := bounds.Min.Y + int(float64(y)*float64(height)/float64(dstHeight))
		for x := 0; x < dstWidth; x++ {
			srcX := bounds.Min.X + int(float64(x)*float64(width)/float64(dstWidth))
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}

func flattenImage(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, bounds.Min, draw.Over)
	return dst
}
