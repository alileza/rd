package extractor

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// facebook handles reels, videos and share links (facebook.com/share/r/...,
// fb.watch/...). Public posts embed progressive MP4 URLs (H.264 + AAC) in the
// page's JSON, so no login or DASH muxing is needed.
type facebook struct{}

func init() { register(facebook{}) }

func (facebook) Name() string { return "facebook" }

func (facebook) Match(u *url.URL) bool {
	return hostIs(u, "facebook.com", "fb.watch", "fb.com")
}

// Keys holding direct MP4 URLs, best first. The browser_native_* keys are used
// by the current web UI, playable_url* by older layouts.
var fbURLKeys = []struct{ key, quality string }{
	{"browser_native_hd_url", "hd"},
	{"playable_url_quality_hd", "hd"},
	{"browser_native_sd_url", "sd"},
	{"playable_url", "sd"},
}

var (
	fbPathID = regexp.MustCompile(`/(?:reel|videos|watch/live)/(?:[^/]+/)?(\d+)`)
	fbPageID = regexp.MustCompile(`"video_id":"(\d+)"`)
	// og:description is the post text; og:title prefixes it with view counts.
	fbOGTitle = []*regexp.Regexp{
		regexp.MustCompile(`<meta property="og:description" content="([^"]*)"`),
		regexp.MustCompile(`<meta property="og:title" content="([^"]*)"`),
	}
)

func (facebook) Extract(ctx context.Context, c *http.Client, u *url.URL) (*Media, error) {
	// m./mbasic./web. serve different markup; the www page is the one we parse.
	if hostIs(u, "facebook.com") {
		cp := *u
		cp.Host = "www.facebook.com"
		u = &cp
	}

	page, final, err := fetchPage(ctx, c, u.String())
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(final.Path, "/login") {
		return nil, errors.New("facebook redirected to login: the post is private or rate-limited")
	}

	id := fbVideoID(final, page)
	for _, k := range fbURLKeys {
		if v := jsonStringNear(page, k.key, id); v != "" {
			return &Media{
				Site:    "facebook",
				ID:      id,
				Title:   fbTitle(page),
				URL:     v,
				Ext:     "mp4",
				Quality: k.quality,
				Headers: map[string]string{"Referer": "https://www.facebook.com/"},
			}, nil
		}
	}
	return nil, errors.New("no video found on the page: the post may be private, require login, or not be a video")
}

func fbVideoID(final *url.URL, page string) string {
	if m := fbPathID.FindStringSubmatch(final.Path); m != nil {
		return m[1]
	}
	if v := final.Query().Get("v"); v != "" {
		return v
	}
	if m := fbPageID.FindStringSubmatch(page); m != nil {
		return m[1]
	}
	return "unknown"
}

func fbTitle(page string) string {
	for _, re := range fbOGTitle {
		if m := re.FindStringSubmatch(page); m != nil {
			t, _, _ := strings.Cut(html.UnescapeString(m[1]), "\n")
			return t
		}
	}
	return ""
}

// jsonStringNear returns the string value of "key": in page. Pages can carry
// several videos (e.g. suggested reels), so when there are multiple matches it
// picks the one closest to the target video's "id" field.
func jsonStringNear(page, key, id string) string {
	needle := `"` + key + `":`
	type hit struct {
		pos int
		val string
	}
	var hits []hit
	for off := 0; ; {
		i := strings.Index(page[off:], needle)
		if i < 0 {
			break
		}
		pos := off + i
		off = pos + len(needle)
		var s *string
		if err := json.NewDecoder(strings.NewReader(page[off:])).Decode(&s); err == nil && s != nil && *s != "" {
			hits = append(hits, hit{pos, *s})
		}
	}
	if len(hits) == 0 {
		return ""
	}
	if len(hits) == 1 || id == "" {
		return hits[0].val
	}

	idNeedle := `"id":"` + id + `"`
	var idPos []int
	for off := 0; ; {
		i := strings.Index(page[off:], idNeedle)
		if i < 0 {
			break
		}
		idPos = append(idPos, off+i)
		off += i + len(idNeedle)
	}
	if len(idPos) == 0 {
		return hits[0].val
	}

	best, bestDist := hits[0].val, -1
	for _, h := range hits {
		for _, p := range idPos {
			d := h.pos - p
			if d < 0 {
				d = -d
			}
			if bestDist < 0 || d < bestDist {
				best, bestDist = h.val, d
			}
		}
	}
	return best
}
