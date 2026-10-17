package platforms

import (
	"fmt"
	"io"
	"net/http"
	"os"
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
