package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/models"
)

// InstagramPlatform publishes image posts, carousels and stories through the Instagram Graph
// API. It needs an Instagram Business/Creator account linked to a Facebook Page
// and a long-lived token (postctl config set instagram.access_token / .account_id).
// The API only accepts publicly reachable image URLs, so post.Images[0] must be https://.
type InstagramPlatform struct {
	token, accountID string
	client           *http.Client
	baseURL          string        // Graph API root; tests point it at httptest
	pollEvery        time.Duration // how often to ask whether a container is processed
	pollFor          time.Duration // give up waiting after this long
}

func NewInstagramPlatform(token, accountID string) *InstagramPlatform {
	return &InstagramPlatform{token: token, accountID: accountID, client: &http.Client{Timeout: 30 * time.Second}, baseURL: "https://graph.facebook.com/v19.0", pollEvery: 2 * time.Second, pollFor: 90 * time.Second}
}

func (i *InstagramPlatform) Name() string { return models.PlatformInstagram }

func (i *InstagramPlatform) IsAuthenticated(ctx context.Context) bool {
	return i.token != "" && i.accountID != ""
}

func (i *InstagramPlatform) Auth(ctx context.Context) error {
	if i.IsAuthenticated(ctx) {
		return nil
	}
	return fmt.Errorf("Instagram uses a token instead of a login. Needs a Business/Creator account linked to a Facebook Page:\n" +
		"  postctl config set instagram.access_token \"LONG_LIVED_TOKEN\"\n" +
		"  postctl config set instagram.account_id \"INSTAGRAM_BUSINESS_ACCOUNT_ID\"\n" +
		"Token scopes: instagram_basic, instagram_content_publish, pages_show_list.")
}

func (i *InstagramPlatform) UploadImage(ctx context.Context, path string) (string, error) {
	return "", nil // the Graph API fetches the image itself from a public URL
}

func (i *InstagramPlatform) graph(ctx context.Context, method, path string, form url.Values) ([]byte, error) {
	if form == nil {
		form = url.Values{}
	}
	form.Set("access_token", i.token)
	var body io.Reader
	u := i.baseURL + path
	if method == http.MethodGet || method == http.MethodDelete {
		u += "?" + form.Encode()
	} else {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := i.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("instagram request failed: %w", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("instagram %s %s failed (status %d): %s", method, path, resp.StatusCode, b)
	}
	return b, nil
}

// container creates a media container and returns its id.
func (i *InstagramPlatform) container(ctx context.Context, form url.Values) (string, error) {
	b, err := i.graph(ctx, http.MethodPost, "/"+i.accountID+"/media", form)
	if err != nil {
		return "", err
	}
	var c struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(b, &c) != nil || c.ID == "" {
		return "", fmt.Errorf("instagram: no container id in response: %s", b)
	}
	return c.ID, i.waitReady(ctx, c.ID)
}

// waitReady polls the container until Instagram has finished processing it;
// publishing earlier fails with "Media ID is not available" (code 9007).
func (i *InstagramPlatform) waitReady(ctx context.Context, id string) error {
	deadline := time.Now().Add(i.pollFor)
	for {
		b, err := i.graph(ctx, http.MethodGet, "/"+url.PathEscape(id), url.Values{"fields": {"status_code,status"}})
		if err != nil {
			return err
		}
		var st struct {
			Code   string `json:"status_code"`
			Detail string `json:"status"`
		}
		_ = json.Unmarshal(b, &st)
		switch st.Code {
		case "FINISHED":
			return nil
		case "ERROR", "EXPIRED":
			return fmt.Errorf("instagram: container %s failed processing (%s): %s", id, st.Code, st.Detail)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("instagram: container %s still %q after %v", id, st.Code, i.pollFor)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(i.pollEvery):
		}
	}
}

// Post publishes post.Images as a feed post: one image, a carousel (2-10 images),
// or a Story when post.Type is "story" (one image, no caption). All image URLs
// must be public https:// URLs.
func (i *InstagramPlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	if !i.IsAuthenticated(ctx) {
		return "", i.Auth(ctx)
	}
	if len(post.Images) == 0 {
		return "", fmt.Errorf("instagram: needs public https:// image URLs in images (text-only posts are not possible)")
	}
	for _, img := range post.Images {
		if !strings.HasPrefix(img, "https://") {
			return "", fmt.Errorf("instagram: image %q is not a public https:// URL", img)
		}
	}
	story := post.Type == "story"
	caption := post.Body
	if !story && len([]rune(caption)) > 2200 {
		return "", fmt.Errorf("instagram: caption has %d characters, limit is 2200", len([]rune(caption)))
	}
	if story && len(post.Images) != 1 {
		return "", fmt.Errorf("instagram: a story takes exactly one image, got %d", len(post.Images))
	}
	if len(post.Images) > 10 {
		return "", fmt.Errorf("instagram: a carousel holds at most 10 images, got %d", len(post.Images))
	}

	var creation string
	var err error
	switch {
	case story:
		creation, err = i.container(ctx, url.Values{"image_url": {post.Images[0]}, "media_type": {"STORIES"}})
	case len(post.Images) == 1:
		creation, err = i.container(ctx, url.Values{"image_url": {post.Images[0]}, "caption": {caption}})
	default:
		var kids []string
		for _, img := range post.Images {
			id, cerr := i.container(ctx, url.Values{"image_url": {img}, "is_carousel_item": {"true"}})
			if cerr != nil {
				return "", cerr
			}
			kids = append(kids, id)
		}
		creation, err = i.container(ctx, url.Values{"media_type": {"CAROUSEL"}, "children": {strings.Join(kids, ",")}, "caption": {caption}})
	}
	if err != nil {
		return "", err
	}

	b, err := i.graph(ctx, http.MethodPost, "/"+i.accountID+"/media_publish", url.Values{"creation_id": {creation}})
	if err != nil {
		return "", err
	}
	var p struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(b, &p) != nil || p.ID == "" {
		return "", fmt.Errorf("instagram: no media id in response: %s", b)
	}
	return p.ID, nil
}

func (i *InstagramPlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	b, err := i.graph(ctx, http.MethodGet, "/"+url.PathEscape(platformID), url.Values{"fields": {"like_count,comments_count"}})
	if err != nil {
		return models.AnalyticsData{}, err
	}
	var r struct {
		Likes    int `json:"like_count"`
		Comments int `json:"comments_count"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return models.AnalyticsData{}, err
	}
	return models.AnalyticsData{PlatformID: platformID, Likes: r.Likes, Comments: r.Comments, FetchedAt: time.Now()}, nil
}

// Delete: the Instagram Graph API has no endpoint to remove media. Say so
// instead of returning nil, otherwise the local record would be dropped while
// the post stays live.
func (i *InstagramPlatform) Delete(ctx context.Context, platformID string) error {
	return fmt.Errorf("instagram: the API cannot delete posts — post %s was not removed (delete it in the app)", platformID)
}
