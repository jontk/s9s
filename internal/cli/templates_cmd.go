package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/jontk/s9s/internal/config"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/fileperms"
	"github.com/jontk/s9s/internal/views"
	"github.com/spf13/cobra"
)

var (
	exportForce bool
	exportDir   string
)

// templatesCmd represents the templates command group
var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "Template management commands",
	Long: `Manage job submission templates.

Templates are merged from three sources (highest to lowest priority):
1. User-saved templates from ~/.s9s/templates/*.json
2. Config YAML templates
3. Built-in hardcoded templates`,
}

// templatesExportCmd represents the templates export command
var templatesExportCmd = &cobra.Command{
	Use:   "export [template-name]",
	Short: "Export templates to disk as JSON files",
	Long: `Export built-in and config templates to ~/.s9s/templates/ as JSON files.

By default, existing files are skipped. Use --force to overwrite.

Examples:
  s9s templates export                    # Export all templates
  s9s templates export "GPU Job"          # Export a specific template
  s9s templates export --force            # Overwrite existing files
  s9s templates export --dir /tmp/tpl     # Export to a custom directory`,
	RunE: runTemplatesExport,
}

// templatesListCmd represents the templates list command
var templatesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available templates",
	Long: `List all available templates merged from all three sources:
built-in, config YAML, and user-saved templates.`,
	RunE: runTemplatesList,
}

func init() {
	templatesExportCmd.Flags().BoolVar(&exportForce, "force", false, "overwrite existing files")
	templatesExportCmd.Flags().StringVar(&exportDir, "dir", "", "output directory (default: ~/.s9s/templates/)")

	templatesCmd.AddCommand(templatesExportCmd)
	templatesCmd.AddCommand(templatesListCmd)
	rootCmd.AddCommand(templatesCmd)
}

// exportTemplate is the JSON structure written to disk, compatible with views.JobTemplate
type exportTemplate struct {
	Name          string             `json:"name"`
	Description   string             `json:"description"`
	JobSubmission *dao.JobSubmission `json:"job_submission"`
}

func runTemplatesExport(_ *cobra.Command, args []string) error {
	cfg, err := config.LoadWithPath(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	templates := getExportableTemplates(cfg)

	// Filter by name if argument provided
	if len(args) > 0 {
		nameFilter := args[0]
		var filtered []*dao.JobTemplate
		for _, t := range templates {
			if t.Name == nameFilter {
				filtered = append(filtered, t)
				break
			}
		}
		if len(filtered) == 0 {
			fmt.Printf("Template %q not found\n", nameFilter)
			return nil
		}
		templates = filtered
	}

	// Determine output directory
	outDir := exportDir
	if outDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to determine home directory: %w", err)
		}
		outDir = filepath.Join(homeDir, ".s9s", "templates")
	}

	// Create output directory if needed
	if err := os.MkdirAll(outDir, fileperms.ConfigDir); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	exported := 0
	for _, t := range templates {
		filename := sanitizeFilename(t.Name)
		outPath := filepath.Join(outDir, filename)

		// Check if file exists and skip unless --force
		if !exportForce {
			if _, err := os.Stat(outPath); err == nil {
				fmt.Printf("Skipped: %q (file exists, use --force to overwrite)\n", t.Name)
				continue
			}
		}

		// Convert to export format with pointer JobSubmission
		js := t.JobSubmission
		et := exportTemplate{
			Name:          t.Name,
			Description:   t.Description,
			JobSubmission: &js,
		}

		data, err := json.MarshalIndent(et, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal template %q: %w", t.Name, err)
		}

		if err := os.WriteFile(outPath, data, fileperms.ConfigFile); err != nil {
			return fmt.Errorf("failed to write template %q: %w", t.Name, err)
		}

		fmt.Printf("Exported: %q -> %s\n", t.Name, outPath)
		exported++
	}

	fmt.Printf("Exported %d templates to %s\n", exported, outDir)
	return nil
}

func runTemplatesList(_ *cobra.Command, _ []string) error {
	cfg, err := config.LoadWithPath(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	type templateEntry struct {
		Name        string
		Source      string
		Description string
	}

	seen := make(map[string]int)
	var entries []templateEntry

	// 1. Built-in templates (lowest priority)
	for _, t := range views.BuiltinTemplates() {
		seen[t.Name] = len(entries)
		entries = append(entries, templateEntry{
			Name:        t.Name,
			Source:      "builtin",
			Description: t.Description,
		})
	}

	// 2. Config YAML templates
	if cfg.Views.Jobs.Submission.Templates != nil {
		for _, ct := range cfg.Views.Jobs.Submission.Templates {
			e := templateEntry{
				Name:        ct.Name,
				Source:      "config",
				Description: ct.Description,
			}
			if idx, ok := seen[ct.Name]; ok {
				entries[idx] = e
			} else {
				seen[ct.Name] = len(entries)
				entries = append(entries, e)
			}
		}
	}

	// 3. Saved templates (highest priority)
	mgr := views.NewJobTemplateManager()
	for _, saved := range mgr.GetTemplates() {
		e := templateEntry{
			Name:        saved.Name,
			Source:      "saved",
			Description: saved.Description,
		}
		if idx, ok := seen[saved.Name]; ok {
			entries[idx] = e
		} else {
			seen[saved.Name] = len(entries)
			entries = append(entries, e)
		}
	}

	if len(entries) == 0 {
		fmt.Println("No templates found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSOURCE\tDESCRIPTION")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\n", e.Name, e.Source, e.Description)
	}
	w.Flush()

	return nil
}

// getExportableTemplates merges built-in and config templates (2 tiers only).
// Saved templates are excluded since they are already on disk.
func getExportableTemplates(cfg *config.Config) []*dao.JobTemplate {
	seen := make(map[string]int)
	var result []*dao.JobTemplate

	// Built-in templates
	for _, t := range views.BuiltinTemplates() {
		seen[t.Name] = len(result)
		result = append(result, t)
	}

	// Config templates override built-in by name
	for _, ct := range cfg.Views.Jobs.Submission.Templates {
		js := views.ConfigValuesToJobSubmission(config.JobSubmissionFromMap(ct.Defaults))
		t := &dao.JobTemplate{
			Name:          ct.Name,
			Description:   ct.Description,
			JobSubmission: js,
		}
		if idx, ok := seen[ct.Name]; ok {
			result[idx] = t
		} else {
			seen[ct.Name] = len(result)
			result = append(result, t)
		}
	}

	return result
}

// sanitizeFilename converts a template name to a safe filename.
// Matches the convention used by JobTemplateManager.saveTemplate().
func sanitizeFilename(name string) string {
	filename := strings.ReplaceAll(name, " ", "_")
	filename = strings.ReplaceAll(filename, "/", "_")
	return strings.ToLower(filename) + ".json"
}
