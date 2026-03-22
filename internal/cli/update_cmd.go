package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jontk/s9s/internal/update"
	"github.com/jontk/s9s/internal/version"
	"github.com/spf13/cobra"
)

var (
	updateCheck      bool
	updateForce      bool
	updatePreRelease bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and install updates",
	Long:  `Check for new versions of s9s and optionally install them.`,
	Example: `  s9s update              # Check and install latest update
  s9s update --check      # Only check for updates
  s9s update --force      # Skip confirmation prompt
  s9s update --pre-release # Include pre-release versions`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().BoolVar(&updateCheck, "check", false, "only check for updates, don't install")
	updateCmd.Flags().BoolVar(&updateForce, "force", false, "skip confirmation prompt")
	updateCmd.Flags().BoolVar(&updatePreRelease, "pre-release", false, "include pre-release versions")
}

func runUpdate(_ *cobra.Command, _ []string) error {
	current := version.Get()
	fmt.Printf("Current version: %s\n", current.Short())

	if current.Version == "dev" {
		return fmt.Errorf("cannot update a development build; install a release from https://github.com/jontk/s9s/releases")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	checker := update.NewChecker()
	release, err := checker.LatestRelease(ctx, updatePreRelease)
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	fmt.Printf("Latest version:  %s\n", release.Version)
	if release.HTMLURL != "" {
		fmt.Printf("Release:         %s\n", release.HTMLURL)
	}

	if !update.IsNewer(current.Version, release.Version) {
		fmt.Println("\nAlready up to date.")
		return nil
	}

	fmt.Printf("\nA new version is available: %s -> %s\n", current.Short(), release.Version)

	if updateCheck {
		return nil
	}

	if !updateForce {
		fmt.Print("\nDo you want to update? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Update cancelled.")
			return nil
		}
	}

	fmt.Printf("\nUpdating s9s %s -> %s...\n", current.Short(), release.Version)

	updater := update.NewUpdater()
	result, err := updater.Update(ctx, update.UpdateOptions{
		PreRelease: updatePreRelease,
		Force:      true, // we already checked IsNewer above
	})
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Printf("Successfully updated to %s!\n", result.NewVersion)
	return nil
}
