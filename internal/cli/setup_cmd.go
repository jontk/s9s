package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard (temporarily disabled)",
	Long:  `The setup wizard is temporarily disabled while being reworked. Please configure s9s manually.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		fmt.Println("The setup wizard is temporarily disabled while being reworked.")
		fmt.Println()
		fmt.Println("To configure s9s manually, create ~/.s9s/config.yaml:")
		fmt.Println()
		fmt.Println("  defaultCluster: my-cluster")
		fmt.Println()
		fmt.Println("  clusters:")
		fmt.Println("    - name: my-cluster")
		fmt.Println("      cluster:")
		fmt.Println("        endpoint: http://your-slurmrestd:6820")
		fmt.Println("        token: ${SLURM_TOKEN}")
		fmt.Println("        apiVersion: v0.0.43")
		fmt.Println()
		fmt.Println("Or run s9s directly on a SLURM node — it will auto-discover slurmrestd.")
		fmt.Println()
		fmt.Println("See https://s9s.dev/docs for full configuration options.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
