package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/sanisideup/jira-cli-for-agents/pkg/config"
	"github.com/sanisideup/jira-cli-for-agents/pkg/template"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage issue templates",
	Long:  `Manage Jira issue templates for creating issues with predefined structures.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Override parent's PersistentPreRunE to skip config loading
		return nil
	},
}

var (
	initLocal bool // --local flag for template init
)

var templateInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize templates by copying defaults to user or project directory",
	Long: `Copies default issue templates (epic, story, bug, charter) to a templates directory
for customization. Existing templates will not be overwritten.

By default, templates are copied to ~/.jcfa/templates/ (user directory).
Use --local to initialize templates in ./.jcfa/templates/ (project directory).`,
	RunE: runTemplateInit,
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available templates",
	Long:  `Lists all available templates from both user directory and defaults.`,
	RunE:  runTemplateList,
}

var templateShowCmd = &cobra.Command{
	Use:   "show <template-name>",
	Short: "Show template contents",
	Long:  `Displays the contents of a specific template.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTemplateShow,
}

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateInitCmd)
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateShowCmd)

	// Add --local flag to template init
	templateInitCmd.Flags().BoolVar(&initLocal, "local", false, "initialize templates in .jcfa/templates/ (project-local)")
}

func runTemplateInit(cmd *cobra.Command, args []string) error {
	var templateDir string

	if initLocal {
		// Initialize in project-local directory
		templateDir = ".jcfa/templates"
	} else {
		// Initialize in user directory (default)
		configDir, err := config.GetConfigDir()
		if err != nil {
			return err
		}
		templateDir = filepath.Join(configDir, "templates")
	}

	svc := template.NewService(templateDir)

	// Initialize templates to the target directory
	if err := svc.InitTemplatesToDir(templateDir); err != nil {
		return fmt.Errorf("failed to initialize templates: %w", err)
	}

	if initLocal {
		fmt.Printf("✓ Templates initialized in project directory: %s\n", templateDir)
		fmt.Println("\nThis directory can be committed to version control.")
	} else {
		fmt.Printf("✓ Templates initialized in: %s\n", templateDir)
	}

	fmt.Println("\nAvailable templates:")
	fmt.Println("  - epic.yaml")
	fmt.Println("  - story.yaml")
	fmt.Println("  - bug.yaml")
	fmt.Println("  - charter.yaml")
	fmt.Printf("\nYou can customize these templates by editing the files in %s\n", templateDir)

	return nil
}

func runTemplateList(cmd *cobra.Command, args []string) error {
	// Create resolver with config templates dir
	var configTemplatesDir string
	loadedCfg := config.LoadOrDefault()
	if loadedCfg != nil {
		configTemplatesDir = loadedCfg.TemplatesDir
	}

	resolver := template.NewResolver(configTemplatesDir, "")
	svc := template.NewServiceWithResolver(resolver)

	// List templates with source info
	templates, err := svc.ListTemplatesWithInfo()
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	if len(templates) == 0 {
		fmt.Println("No templates found. Run 'jcfa template init' to initialize default templates.")
		return nil
	}

	if jsonOutput {
		data, err := json.MarshalIndent(templates, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Println("Available templates:")
	for _, tmpl := range templates {
		fmt.Printf("  - %s (%s)\n", tmpl.Name, tmpl.Source)
	}

	return nil
}

func runTemplateShow(cmd *cobra.Command, args []string) error {
	templateName := args[0]

	// Create resolver with config templates dir
	var configTemplatesDir string
	loadedCfg := config.LoadOrDefault()
	if loadedCfg != nil {
		configTemplatesDir = loadedCfg.TemplatesDir
	}

	resolver := template.NewResolver(configTemplatesDir, "")
	svc := template.NewServiceWithResolver(resolver)

	// Resolve template path first to show source
	path, source, resolveErr := resolver.ResolveWithBuiltin(templateName, svc.GetBuiltinNames)
	
	// Load template
	tmpl, err := svc.LoadTemplate(templateName)
	if err != nil {
		return err
	}

	if jsonOutput {
		// Include path and source in JSON output
		output := map[string]interface{}{
			"name":   templateName,
			"path":   path,
			"source": source,
			"type":   tmpl.Type,
			"fields": tmpl.Fields,
		}
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Template: %s\n", templateName)
	if resolveErr == nil {
		if source == "builtin" {
			fmt.Printf("Path: (builtin)\n")
		} else {
			fmt.Printf("Path: %s\n", path)
		}
		fmt.Printf("Source: %s\n", source)
	}
	fmt.Printf("Issue Type: %s\n", tmpl.Type)
	fmt.Println("\nFields:")

	// Pretty print fields
	for fieldKey, fieldValue := range tmpl.Fields {
		valueJSON, _ := json.MarshalIndent(fieldValue, "  ", "  ")
		fmt.Printf("  %s: %s\n", fieldKey, string(valueJSON))
	}

	return nil
}
