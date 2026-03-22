package update

import (
	"testing"

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
