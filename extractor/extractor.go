// Package extractor resolves a social media post URL into a direct media URL.
//
// Each site lives in its own file and registers itself via register() in an
// init function. Adding a new site (Instagram, YouTube Shorts, ...) means adding
// one file that implements Extractor.
package extractor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// UserAgent is a desktop browser UA; most sites serve a stripped-down or
// login-walled page to unknown clients.
const UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"

// Media is a single downloadable file.
type Media struct {
	Site    string
	ID      string
	Title   string
	URL     string
	Ext     string
	Quality string
	// Headers to send when downloading URL (e.g. Referer).
	Headers map[string]string
}

// Filename is the default local file name for the media.
func (m *Media) Filename() string {
	return fmt.Sprintf("%s-%s.%s", m.Site, m.ID, m.Ext)
}

type Extractor interface {
	Name() string
	Match(u *url.URL) bool
	Extract(ctx context.Context, c *http.Client, u *url.URL) (*Media, error)
}

var ErrUnsupported = errors.New("unsupported URL")

var registry []Extractor

func register(e Extractor) { registry = append(registry, e) }

// Find returns the extractor that handles rawURL.
func Find(rawURL string) (Extractor, *url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, nil, fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme == "" {
		// Allow "facebook.com/reel/..." without a scheme.
		if u, err = url.Parse("https://" + strings.TrimSpace(rawURL)); err != nil {
			return nil, nil, fmt.Errorf("parse url: %w", err)
		}
	}
	for _, e := range registry {
		if e.Match(u) {
			return e, u, nil
		}
	}
	return nil, nil, fmt.Errorf("%w: %s", ErrUnsupported, u.Host)
}

// NewClient returns an HTTP client suitable for scraping.
func NewClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

// fetchPage GETs a page with browser-like headers and returns the body and the
// final URL after redirects.
func fetchPage(ctx context.Context, c *http.Client, u string) (string, *url.URL, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Site", "none")

	resp, err := c.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("GET %s: %s", u, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}
	return string(body), resp.Request.URL, nil
}

func hostIs(u *url.URL, domains ...string) bool {
	h := strings.ToLower(u.Hostname())
	for _, d := range domains {
		if h == d || strings.HasSuffix(h, "."+d) {
			return true
		}
	}
	return false
}
