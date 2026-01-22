package cli

import (
	"fmt"

	"github.com/jontk/s9s/internal/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version information for s9s including build details.`,
	RunE:  runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) error {
	info := version.Get()
	fmt.Print(info.Full())
	fmt.Println() // Add final newline
	return nil
}
