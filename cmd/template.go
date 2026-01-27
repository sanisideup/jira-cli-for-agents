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

var templateInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize templates by copying defaults to user directory",
	Long: `Copies default issue templates (epic, story, bug, charter) to ~/.jcfa/templates/
for customization. Existing templates will not be overwritten.`,
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
}

func runTemplateInit(cmd *cobra.Command, args []string) error {
	// Get template directory
	configDir, err := config.GetConfigDir()
	if err != nil {
		return err
	}

	templateDir := filepath.Join(configDir, "templates")
	svc := template.NewService(templateDir)

	// Initialize templates
	if err := svc.InitTemplates(); err != nil {
		return fmt.Errorf("failed to initialize templates: %w", err)
	}

	fmt.Printf("✓ Templates initialized in: %s\n", templateDir)
	fmt.Println("\nAvailable templates:")
	fmt.Println("  - epic.yaml")
	fmt.Println("  - story.yaml")
	fmt.Println("  - bug.yaml")
	fmt.Println("  - charter.yaml")
	fmt.Printf("\nYou can customize these templates by editing the files in %s\n", templateDir)

	return nil
}

func runTemplateList(cmd *cobra.Command, args []string) error {
	// Get template directory
	configDir, err := config.GetConfigDir()
	if err != nil {
		return err
	}

	templateDir := filepath.Join(configDir, "templates")
	svc := template.NewService(templateDir)

	// List templates
	templates, err := svc.ListTemplates()
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
	for _, name := range templates {
		fmt.Printf("  - %s\n", name)
	}

	return nil
}

func runTemplateShow(cmd *cobra.Command, args []string) error {
	templateName := args[0]

	// Get template directory
	configDir, err := config.GetConfigDir()
	if err != nil {
		return err
	}

	templateDir := filepath.Join(configDir, "templates")
	svc := template.NewService(templateDir)

	// Load template
	tmpl, err := svc.LoadTemplate(templateName)
	if err != nil {
		return err
	}

	if jsonOutput {
		data, err := json.MarshalIndent(tmpl, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Template: %s\n", templateName)
	fmt.Printf("Issue Type: %s\n", tmpl.Type)
	fmt.Println("\nFields:")

	// Pretty print fields
	for fieldKey, fieldValue := range tmpl.Fields {
		valueJSON, _ := json.MarshalIndent(fieldValue, "  ", "  ")
		fmt.Printf("  %s: %s\n", fieldKey, string(valueJSON))
	}

	return nil
}
