// Package version provides version information for s9s
package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version (set by goreleaser at build time)
	Version = "dev"

	// Commit is the git commit hash (set by goreleaser at build time)
	Commit = "unknown"

	// Date is the build date (set by goreleaser at build time)
	Date = "unknown"

	// BuiltBy indicates who built the binary (set by goreleaser at build time)
	BuiltBy = "unknown"
)

// Info holds version information
type Info struct {
	Version   string
	Commit    string
	BuildDate string
	BuiltBy   string
	GoVersion string
	Platform  string
}

// Get returns version information
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: Date,
		BuiltBy:   BuiltBy,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a human-readable version string
func (i *Info) String() string {
	if i.Commit != "unknown" && len(i.Commit) > 7 {
		return fmt.Sprintf("%s (commit: %s, built: %s)", i.Version, i.Commit[:7], i.BuildDate)
	}
	return fmt.Sprintf("%s (built: %s)", i.Version, i.BuildDate)
}

// Short returns just the version number
func (i *Info) Short() string {
	return i.Version
}

// Full returns detailed version information
func (i *Info) Full() string {
	return fmt.Sprintf(`s9s version %s
Git commit: %s
Built: %s
Built by: %s
Go version: %s
Platform: %s`,
		i.Version, i.Commit, i.BuildDate, i.BuiltBy, i.GoVersion, i.Platform)
}
