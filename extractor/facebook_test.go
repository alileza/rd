package extractor

import "testing"

func TestJSONStringNear(t *testing.T) {
	page := `{"video":{"id":"111","browser_native_hd_url":"https:\/\/cdn\/a.mp4?x=1%3D"}}` +
		`{"video":{"id":"222","browser_native_hd_url":null}}` +
		`{"video":{"browser_native_hd_url":"https:\/\/cdn\/b.mp4","id":"333"}}`

	tests := []struct {
		id, want string
	}{
		{"111", "https://cdn/a.mp4?x=1%3D"},
		{"333", "https://cdn/b.mp4"},
		{"", "https://cdn/a.mp4?x=1%3D"},
		{"999", "https://cdn/a.mp4?x=1%3D"},
	}
	for _, tt := range tests {
		if got := jsonStringNear(page, "browser_native_hd_url", tt.id); got != tt.want {
			t.Errorf("id=%q: got %q, want %q", tt.id, got, tt.want)
		}
	}
	if got := jsonStringNear(page, "playable_url", "111"); got != "" {
		t.Errorf("missing key: got %q", got)
	}
}
