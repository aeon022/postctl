package platforms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
)

// linkedinAPIVersion is the Linkedin-Version header (YYYYMM) sent to the versioned
// /rest APIs. Versions stay supported for about a year; bump when LinkedIn sunsets it.
const linkedinAPIVersion = "202606"

// orgMode reports whether posts go out as a company page (Posts API, needs the
// Community Management API product) instead of as the logged-in person (ugcPosts).
func orgMode() bool { return config.ActiveConfig.LinkedIn.OrganizationID != "" }

func (l *LinkedInPlatform) restRequest(ctx context.Context, method, path, token string, body []byte) (*http.Request, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, l.apiURL+path, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")
	req.Header.Set("Linkedin-Version", linkedinAPIVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// imageBytes reads a local file, or downloads it when it is an https:// URL.
func (l *LinkedInPlatform) imageBytes(ctx context.Context, src string) ([]byte, error) {
	if !strings.HasPrefix(src, "https://") {
		return os.ReadFile(src)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return nil, err
	}
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: status %d", src, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// uploadImageREST registers and uploads an image through the Images API and returns its urn:li:image.
func (l *LinkedInPlatform) uploadImageREST(ctx context.Context, token, owner, src string) (string, error) {
	data, err := l.imageBytes(ctx, src)
	if err != nil {
		return "", fmt.Errorf("read image: %w", err)
	}
	body, _ := json.Marshal(map[string]any{"initializeUploadRequest": map[string]string{"owner": owner}})
	req, err := l.restRequest(ctx, http.MethodPost, "/rest/images?action=initializeUpload", token, body)
	if err != nil {
		return "", err
	}
	resp, err := l.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("initialize image upload: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("initialize image upload failed (status %d): %s", resp.StatusCode, raw)
	}
	var init struct {
		Value struct {
			UploadURL string `json:"uploadUrl"`
			Image     string `json:"image"`
		} `json:"value"`
	}
	if json.Unmarshal(raw, &init) != nil || init.Value.UploadURL == "" || init.Value.Image == "" {
		return "", fmt.Errorf("initialize image upload: unexpected response: %s", raw)
	}

	put, err := http.NewRequestWithContext(ctx, http.MethodPut, init.Value.UploadURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	put.Header.Set("Authorization", "Bearer "+token)
	put.Header.Set("Content-Type", "application/octet-stream")
	presp, err := l.client.Do(put)
	if err != nil {
		return "", fmt.Errorf("upload image: %w", err)
	}
	defer presp.Body.Close()
	if presp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(presp.Body, 1024))
		return "", fmt.Errorf("upload image failed (status %d): %s", presp.StatusCode, b)
	}
	return init.Value.Image, nil
}

// postOrg publishes as the company page via the Posts API (POST /rest/posts).
// Text or a single image; the post id comes back in the x-restli-id header.
func (l *LinkedInPlatform) postOrg(ctx context.Context, token string, post *models.Post) (string, error) {
	author := "urn:li:organization:" + config.ActiveConfig.LinkedIn.OrganizationID
	if len(post.Images) > 1 {
		return "", fmt.Errorf("linkedin: company-page posts with several images are not supported yet (got %d)", len(post.Images))
	}
	body := map[string]any{
		"author":                    author,
		"commentary":                post.Body,
		"visibility":                "PUBLIC",
		"distribution":              map[string]any{"feedDistribution": "MAIN_FEED", "targetEntities": []any{}, "thirdPartyDistributionChannels": []any{}},
		"lifecycleState":            "PUBLISHED",
		"isReshareDisabledByAuthor": false,
	}
	if len(post.Images) == 1 {
		img, err := l.uploadImageREST(ctx, token, author, post.Images[0])
		if err != nil {
			return "", err
		}
		media := map[string]string{"id": img}
		if post.Title != "" {
			media["altText"] = post.Title
		}
		body["content"] = map[string]any{"media": media}
	}
	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := l.restRequest(ctx, http.MethodPost, "/rest/posts", token, b)
	if err != nil {
		return "", err
	}
	resp, err := l.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("linkedin: post request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("linkedin company-page post failed (status %d): %s", resp.StatusCode, raw)
	}
	id := resp.Header.Get("x-restli-id")
	if id == "" {
		return "", fmt.Errorf("linkedin: post created but no x-restli-id header in response")
	}
	return id, nil
}

// deleteOrg removes a post through the Posts API (204 on success, idempotent).
func (l *LinkedInPlatform) deleteOrg(ctx context.Context, token, id string) error {
	req, err := l.restRequest(ctx, http.MethodDelete, "/rest/posts/"+url.PathEscape(id), token, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-RestLi-Method", "DELETE")
	return doDelete(l.client, req, "linkedin")
}
