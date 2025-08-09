package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// These will be set by the build process
	Version   = "0.1.0-dev"
	CommitSHA = "unknown"
	BuildTime = "unknown"
	GoVersion = runtime.Version()
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information", 
	Long: `Display version information for s9s including build details.`,
	RunE: runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) error {
	fmt.Printf("s9s version %s\n", Version)
	fmt.Printf("Git commit: %s\n", CommitSHA)  
	fmt.Printf("Built: %s\n", BuildTime)
	fmt.Printf("Go version: %s\n", GoVersion)
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	
	return nil
}