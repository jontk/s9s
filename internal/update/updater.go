package update

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/creativeprojects/go-selfupdate"

	"github.com/jontk/s9s/internal/version"
)

// UpdateOptions configures the self-update behavior.
type UpdateOptions struct {
	PreRelease    bool
	Force         bool   // skip "already up to date" check
	TargetVersion string // update to a specific version (e.g. "v0.7.0"); empty = latest
}

// UpdateResult contains the outcome of an update operation.
type UpdateResult struct {
	PreviousVersion string
	NewVersion      string
	ReleaseURL      string
}

// Updater performs the self-update of the s9s binary.
type Updater struct {
	owner   string
	repo    string
	current string
}

// NewUpdater creates an Updater with the current build version.
func NewUpdater() *Updater {
	return &Updater{
		owner:   defaultOwner,
		repo:    defaultRepo,
		current: version.Version,
	}
}

// CanUpdate checks whether the running binary can be updated.
// go-selfupdate writes to a temp file then renames over the binary,
// so we check that the parent directory is writable (not the binary itself,
// which would fail with ETXTBSY on Linux).
func CanUpdate() error {
	if version.Version == "dev" {
		return fmt.Errorf("cannot update a development build; install a release version first")
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}

	if _, err := os.Stat(exe); err != nil {
		return fmt.Errorf("cannot stat executable: %w", err)
	}

	// Check that the directory containing the binary is writable,
	// which is what's needed for the atomic rename.
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".s9s-update-check-*")
	if err != nil {
		return fmt.Errorf("directory %s is not writable (try sudo): %w", dir, err)
	}
	tmp.Close()
	os.Remove(tmp.Name())

	return nil
}

// Update downloads and installs the latest release, replacing the running binary.
func (u *Updater) Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error) {
	if err := CanUpdate(); err != nil {
		return nil, err
	}

	source, err := goSelfupdateSource()
	if err != nil {
		return nil, fmt.Errorf("creating update source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Filters:  assetFilters(),
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
	})
	if err != nil {
		return nil, fmt.Errorf("creating updater: %w", err)
	}

	repo := selfupdate.NewRepositorySlug(u.owner, u.repo)

	var latest *selfupdate.Release
	var found bool

	if opts.TargetVersion != "" {
		latest, found, err = updater.DetectVersion(ctx, repo, ensureVPrefix(opts.TargetVersion))
	} else {
		latest, found, err = updater.DetectLatest(ctx, repo)
	}
	if err != nil {
		return nil, fmt.Errorf("detecting release: %w", err)
	}

	if !found {
		if opts.TargetVersion != "" {
			return nil, fmt.Errorf("release %s not found for %s/%s", opts.TargetVersion, runtime.GOOS, runtime.GOARCH)
		}
		return nil, fmt.Errorf("no release found for %s/%s on %s", runtime.GOOS, runtime.GOARCH, u.repo)
	}

	currentVer := ensureVPrefix(u.current)
	if !opts.Force && !latest.GreaterThan(currentVer) {
		return nil, fmt.Errorf("already up to date (%s)", u.current)
	}

	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("cannot determine executable path: %w", err)
	}

	if err := updater.UpdateTo(ctx, latest, exe); err != nil {
		return nil, fmt.Errorf("applying update: %w", err)
	}

	return &UpdateResult{
		PreviousVersion: u.current,
		NewVersion:      latest.Version(),
		ReleaseURL:      latest.ReleaseNotes,
	}, nil
}

func goSelfupdateSource() (selfupdate.Source, error) {
	return selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
}

func assetFilters() []string {
	// go-selfupdate handles OS/arch matching from archive names automatically.
	// Return nil to use defaults which match the goreleaser naming pattern.
	return nil
}
