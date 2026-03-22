package app

import (
	"context"
	"fmt"
	"time"

	"github.com/jontk/s9s/internal/update"
	"github.com/jontk/s9s/internal/version"
)

// checkForUpdates runs a background update check and notifies via status bar.
func (s *S9s) checkForUpdates() {
	if version.Version == "dev" {
		return
	}

	interval, err := time.ParseDuration(s.config.Update.CheckInterval)
	if err != nil {
		interval = 24 * time.Hour
	}

	state := update.LoadState()

	// If we've checked recently, use cached result.
	if !state.NeedsCheck(interval) {
		if state.LatestVersion != "" && update.IsNewer(version.Version, state.LatestVersion) {
			s.notifyUpdateAvailable(state.LatestVersion)
		}
		return
	}

	// Use a short timeout for the background check so air-gapped clusters
	// don't waste startup time waiting for a network request to timeout.
	ctx, cancel := context.WithTimeout(s.ctx, 3*time.Second)
	defer cancel()

	checker := update.NewChecker()
	release, err := checker.LatestRelease(ctx, s.config.Update.PreRelease)
	if err != nil {
		s.logger.Debug().Err(err).Msg("Background update check failed")
		return
	}

	// Save state regardless of whether update is available.
	state.LastChecked = time.Now()
	state.LatestVersion = release.Version
	state.LatestURL = release.HTMLURL
	state.CurrentVersion = version.Version
	if err := state.Save(); err != nil {
		s.logger.Debug().Err(err).Msg("Failed to save update state")
	}

	if update.IsNewer(version.Version, release.Version) {
		s.notifyUpdateAvailable(release.Version)
	}
}

func (s *S9s) notifyUpdateAvailable(latestVersion string) {
	msg := fmt.Sprintf("Update available: %s -> %s (run 's9s update')", version.Version, latestVersion)
	s.app.QueueUpdateDraw(func() {
		s.statusBar.Info(msg)
	})
}
