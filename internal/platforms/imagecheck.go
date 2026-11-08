package platforms

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aeon022/postctl/internal/models"
)

const maxImageBytes = 25 << 20

// CheckLocalImage refuses anything that is not a regular image file. Without it a post that lists
// e.g. ~/.ssh/id_ed25519 under images: would upload that file to a public platform.
// https/http URLs are fetched by the platform itself and pass.
func CheckLocalImage(path string) error {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return nil
	}
	fi, err := os.Stat(path) // follows symlinks: the target is what gets read
	if err != nil {
		return fmt.Errorf("image %q: %w", path, err)
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("image %q is not a regular file", path)
	}
	if fi.Size() > maxImageBytes {
		return fmt.Errorf("image %q is larger than %d MB", path, maxImageBytes>>20)
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("image %q: %w", path, err)
	}
	defer f.Close()
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	switch http.DetectContentType(head[:n]) {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return nil
	}
	return fmt.Errorf("%q is not a PNG/JPEG/GIF/WebP image — refusing to upload it", path)
}

// CheckPostImages validates every local image of a post (single posts and thread tweets).
func CheckPostImages(p *models.Post) error {
	for _, img := range p.Images {
		if err := CheckLocalImage(img); err != nil {
			return err
		}
	}
	for _, t := range p.Tweets {
		if t.Image != "" {
			if err := CheckLocalImage(t.Image); err != nil {
				return err
			}
		}
	}
	return nil
}

// openImage opens a local image, or downloads an http(s) image (platforms like X need the bytes,
// not a URL). Returns the content and a filename for the multipart upload.
func openImage(ctx context.Context, path string) (io.ReadCloser, string, error) {
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		f, err := os.Open(path)
		return f, filepath.Base(path), err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("download image %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download image %s: status %d", path, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("download image %s: %w", path, err)
	}
	if len(data) > maxImageBytes {
		return nil, "", fmt.Errorf("image %s is larger than %d MB", path, maxImageBytes>>20)
	}
	switch http.DetectContentType(data) {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
	default:
		return nil, "", fmt.Errorf("%s is not a PNG/JPEG/GIF/WebP image", path)
	}
	name := "image"
	if u, err := url.Parse(path); err == nil {
		name = filepath.Base(u.Path)
	}
	return io.NopCloser(bytes.NewReader(data)), name, nil
}
