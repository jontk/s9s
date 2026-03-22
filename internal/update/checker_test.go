package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{"newer patch", "v0.7.1", "v0.7.2", true},
		{"newer minor", "v0.7.1", "v0.8.0", true},
		{"newer major", "v0.7.1", "v1.0.0", true},
		{"same version", "v0.7.1", "v0.7.1", false},
		{"older version", "v0.8.0", "v0.7.1", false},
		{"dev build", "dev", "v0.7.1", false},
		{"empty current", "", "v0.7.1", false},
		{"empty latest", "v0.7.1", "", false},
		{"without v prefix", "0.7.1", "0.7.2", true},
		{"mixed prefix", "v0.7.1", "0.7.2", true},
		{"invalid current", "notaversion", "v0.7.1", false},
		{"invalid latest", "v0.7.1", "notaversion", false},
		{"pre-release newer", "v0.7.1", "v0.8.0-rc.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNewer(tt.current, tt.latest)
			if got != tt.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

// newTestChecker creates a Checker pointed at the given test server.
func newTestChecker(srv *httptest.Server) *Checker {
	return &Checker{
		owner:      "testowner",
		repo:       "testrepo",
		baseURL:    srv.URL,
		httpClient: srv.Client(),
	}
}

func TestChecker_LatestRelease_Stable(t *testing.T) {
	release := ghRelease{
		TagName:     "v0.8.0",
		Name:        "v0.8.0",
		PreRelease:  false,
		PublishedAt: time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
		HTMLURL:     "https://github.com/jontk/s9s/releases/tag/v0.8.0",
		Assets: []ghAsset{
			{
				Name:               "s9s_0.8.0_Linux_x86_64.tar.gz",
				BrowserDownloadURL: "https://example.com/s9s_0.8.0_Linux_x86_64.tar.gz",
				Size:               10_000_000,
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/testowner/testrepo/releases/latest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(release)
	}))
	defer srv.Close()

	checker := newTestChecker(srv)
	info, err := checker.LatestRelease(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Version != "0.8.0" {
		t.Errorf("Version = %q, want %q", info.Version, "0.8.0")
	}
	if info.TagName != "v0.8.0" {
		t.Errorf("TagName = %q, want %q", info.TagName, "v0.8.0")
	}
	if len(info.Assets) != 1 {
		t.Errorf("Assets count = %d, want 1", len(info.Assets))
	}
}

func TestChecker_LatestRelease_PreRelease(t *testing.T) {
	releases := []ghRelease{
		{
			TagName:     "v0.9.0-rc.1",
			PreRelease:  true,
			PublishedAt: time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC),
			HTMLURL:     "https://github.com/jontk/s9s/releases/tag/v0.9.0-rc.1",
		},
		{
			TagName:     "v0.8.0",
			PreRelease:  false,
			PublishedAt: time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
			HTMLURL:     "https://github.com/jontk/s9s/releases/tag/v0.8.0",
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(releases)
	}))
	defer srv.Close()

	checker := newTestChecker(srv)
	info, err := checker.LatestRelease(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.TagName != "v0.9.0-rc.1" {
		t.Errorf("TagName = %q, want %q", info.TagName, "v0.9.0-rc.1")
	}
}

func TestChecker_LatestRelease_FiltersDrafts(t *testing.T) {
	releases := []ghRelease{
		{
			TagName:    "v0.9.0",
			Draft:      true,
			PreRelease: false,
		},
		{
			TagName:    "v0.8.0",
			Draft:      false,
			PreRelease: false,
			HTMLURL:    "https://github.com/jontk/s9s/releases/tag/v0.8.0",
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(releases)
	}))
	defer srv.Close()

	checker := newTestChecker(srv)
	info, err := checker.LatestRelease(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should skip the draft v0.9.0 and return v0.8.0
	if info.TagName != "v0.8.0" {
		t.Errorf("TagName = %q, want %q (draft should be filtered)", info.TagName, "v0.8.0")
	}
}

func TestChecker_LatestRelease_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	checker := newTestChecker(srv)
	_, err := checker.LatestRelease(context.Background(), false)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestChecker_LatestRelease_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	checker := newTestChecker(srv)
	_, err := checker.LatestRelease(context.Background(), false)
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
}
