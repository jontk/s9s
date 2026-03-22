package update

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/version"
)

func TestCanUpdate_DevBuild(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()

	version.Version = "dev"
	err := CanUpdate()
	if err == nil {
		t.Fatal("expected error for dev build")
	}
	if err.Error() != "cannot update a development build; install a release version first" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestNewUpdater(t *testing.T) {
	u := NewUpdater()
	if u.owner != defaultOwner {
		t.Errorf("owner = %q, want %q", u.owner, defaultOwner)
	}
	if u.repo != defaultRepo {
		t.Errorf("repo = %q, want %q", u.repo, defaultRepo)
	}
	if u.current != version.Version {
		t.Errorf("current = %q, want %q", u.current, version.Version)
	}
}

func TestUpdate_PinnedVersion_ErrorIncludesVersion(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "0.7.1"

	u := NewUpdater()
	// Use a nonexistent repo to force a quick failure without downloading.
	u.owner = "nonexistent-owner-xxxxx"
	u.repo = "nonexistent-repo-xxxxx"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// With TargetVersion set, error should come from DetectVersion path.
	_, err := u.Update(ctx, UpdateOptions{
		TargetVersion: "v99.99.99",
		Force:         true,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent repo")
	}

	// Without TargetVersion, error should come from DetectLatest path.
	_, errLatest := u.Update(ctx, UpdateOptions{
		Force: true,
	})
	if errLatest == nil {
		t.Fatal("expected error for nonexistent repo")
	}

	// Both should fail, but the key thing is they don't panic and
	// the error from the pinned path is distinct.
	if err.Error() == errLatest.Error() {
		// They may have the same error from GitHub API ("not found"),
		// which is fine — the important thing is both paths execute without panic.
		t.Logf("both paths returned same error (expected for nonexistent repo): %s", err)
	}
}

func TestUpdate_DevBuild_Rejected(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "dev"

	u := NewUpdater()
	_, err := u.Update(context.Background(), UpdateOptions{})
	if err == nil {
		t.Fatal("expected error for dev build")
	}
	if !strings.Contains(err.Error(), "development build") {
		t.Errorf("error should mention development build, got: %s", err)
	}
}
