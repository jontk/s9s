// Package update provides version checking and self-update functionality for s9s.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	"github.com/jontk/s9s/internal/version"
)

const (
	defaultOwner = "jontk"
	defaultRepo  = "s9s"
)

const defaultBaseURL = "https://api.github.com"

// ReleaseInfo holds information about a GitHub release.
type ReleaseInfo struct {
	Version     string
	TagName     string
	PublishedAt time.Time
	PreRelease  bool
	HTMLURL     string
	Assets      []Asset
}

// Asset represents a downloadable release artifact.
type Asset struct {
	Name        string
	DownloadURL string
	Size        int64
}

// Checker queries GitHub for new releases.
type Checker struct {
	owner      string
	repo       string
	baseURL    string
	httpClient *http.Client
}

// NewChecker creates a Checker for the default s9s repository.
func NewChecker() *Checker {
	return NewCheckerFor(defaultOwner, defaultRepo)
}

// NewCheckerFor creates a Checker for a specific GitHub repository.
func NewCheckerFor(owner, repo string) *Checker {
	return &Checker{
		owner:   owner,
		repo:    repo,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// LatestRelease fetches the latest release from GitHub.
// If includePreRelease is true, pre-release versions are considered.
func (c *Checker) LatestRelease(ctx context.Context, includePreRelease bool) (*ReleaseInfo, error) {
	if includePreRelease {
		return c.latestIncludingPreRelease(ctx)
	}
	return c.latestStable(ctx)
}

func (c *Checker) latestStable(ctx context.Context) (*ReleaseInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", c.baseURL, c.owner, c.repo)

	var gh ghRelease
	if err := c.fetchJSON(ctx, url, &gh); err != nil {
		return nil, fmt.Errorf("fetching latest release: %w", err)
	}

	return gh.toReleaseInfo(), nil
}

func (c *Checker) latestIncludingPreRelease(ctx context.Context) (*ReleaseInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=10", c.baseURL, c.owner, c.repo)

	var releases []ghRelease
	if err := c.fetchJSON(ctx, url, &releases); err != nil {
		return nil, fmt.Errorf("fetching releases: %w", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found")
	}

	// Find the newest non-draft release by semver.
	var best *ghRelease
	for i := range releases {
		r := &releases[i]
		if r.Draft {
			continue
		}
		if best == nil {
			best = r
			continue
		}
		if semver.Compare(ensureVPrefix(r.TagName), ensureVPrefix(best.TagName)) > 0 {
			best = r
		}
	}

	if best == nil {
		return nil, fmt.Errorf("no non-draft releases found")
	}

	return best.toReleaseInfo(), nil
}

func (c *Checker) fetchJSON(ctx context.Context, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("s9s/%s", version.Version))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("GitHub API rate limit exceeded (status %d)", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from GitHub API", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}

// IsNewer returns true if latest is a newer semver than current.
// Returns false for dev builds or invalid versions.
func IsNewer(current, latest string) bool {
	if current == "dev" || current == "" || latest == "" {
		return false
	}

	c := ensureVPrefix(current)
	l := ensureVPrefix(latest)

	if !semver.IsValid(c) || !semver.IsValid(l) {
		return false
	}

	return semver.Compare(l, c) > 0
}

func ensureVPrefix(v string) string {
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}

// ghRelease is the GitHub API release response.
type ghRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PreRelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Assets      []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

func (r *ghRelease) toReleaseInfo() *ReleaseInfo {
	assets := make([]Asset, len(r.Assets))
	for i, a := range r.Assets {
		assets[i] = Asset{
			Name:        a.Name,
			DownloadURL: a.BrowserDownloadURL,
			Size:        a.Size,
		}
	}

	ver := strings.TrimPrefix(r.TagName, "v")

	return &ReleaseInfo{
		Version:     ver,
		TagName:     r.TagName,
		PublishedAt: r.PublishedAt,
		PreRelease:  r.PreRelease,
		HTMLURL:     r.HTMLURL,
		Assets:      assets,
	}
}
