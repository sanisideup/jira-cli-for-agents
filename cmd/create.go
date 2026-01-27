package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sanisideup/jira-cli-for-agents/pkg/jira"
	"github.com/sanisideup/jira-cli-for-agents/pkg/template"
	"github.com/spf13/cobra"
)

var (
	templateName string
	dataFile     string
	dryRun       bool
	interactive  bool
	parentIssue  string // Parent issue key for creating subtasks
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a single Jira issue",
	Long: `Create a single Jira issue using a template and data file.

Subtask Creation:
  Use --parent to create a subtask under an existing issue.
  The parent issue must exist and cannot itself be a subtask.
  The issue type should typically be "Sub-task" or a similar subtask type.

Examples:
  # Create from template and data file
  jcfa create --template story --data story.json

  # Create with dry-run to validate
  jcfa create --template story --data story.json --dry-run

  # Create with JSON output
  jcfa create --template story --data story.json --json

  # Read data from stdin
  cat story.json | jcfa create --template story --data -

  # Create a subtask under a parent issue
  jcfa create --template subtask --data task.json --parent PROJ-123

  # Create subtask interactively
  jcfa create --template subtask --interactive --parent PROJ-123
`,
	RunE: runCreate,
}

func init() {
	rootCmd.AddCommand(createCmd)

	createCmd.Flags().StringVarP(&templateName, "template", "t", "", "template name to use (required)")
	createCmd.Flags().StringVarP(&dataFile, "data", "d", "", "JSON file with template data (use '-' for stdin)")
	createCmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate without creating the issue")
	createCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "interactive mode (prompts for data)")
	createCmd.Flags().StringVarP(&parentIssue, "parent", "p", "", "parent issue key for creating subtasks (e.g., PROJ-123)")

	createCmd.MarkFlagRequired("template")
}

func runCreate(cmd *cobra.Command, args []string) error {
	// Validate flags
	if !interactive && dataFile == "" {
		return fmt.Errorf("either --data or --interactive must be specified")
	}

	// Initialize services
	templateService := template.NewService(filepath.Join(os.Getenv("HOME"), ".jcfa", "templates"))
	issueService := jira.NewIssueService(jiraClient)

	// Load template
	tmpl, err := templateService.LoadTemplate(templateName)
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	// Get template data
	var data map[string]interface{}
	if interactive {
		// Interactive mode: prompt user for data
		data, err = promptForTemplateData(tmpl)
		if err != nil {
			return fmt.Errorf("failed to get template data: %w", err)
		}
	} else {
		// Load from file or stdin
		data, err = loadTemplateData(dataFile)
		if err != nil {
			return fmt.Errorf("failed to load template data: %w", err)
		}
	}

	// Render template
	fields, err := templateService.RenderTemplate(tmpl, data, cfg)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Ensure required fields are present
	if err := ensureRequiredFields(fields, tmpl); err != nil {
		return err
	}

	// Handle subtask creation if --parent is specified
	if parentIssue != "" {
		if err := setupParentIssue(fields, parentIssue, issueService); err != nil {
			return err
		}
	}

	// Validate issue fields
	if err := issueService.ValidateIssueFields(fields); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Dry run: just show what would be created
	if dryRun {
		if jsonOutput {
			output, _ := json.MarshalIndent(fields, "", "  ")
			fmt.Println(string(output))
		} else {
			fmt.Println("✓ Validation passed. Would create issue with fields:")
			printFields(fields)
			if parentIssue != "" {
				fmt.Printf("\n  (Will be created as subtask of %s)\n", parentIssue)
			}
		}
		return nil
	}

	// Create the issue
	result, err := issueService.CreateIssue(fields)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	// Output result
	if jsonOutput {
		outputData := map[string]interface{}{
			"id":   result.ID,
			"key":  result.Key,
			"self": result.Self,
		}
		if parentIssue != "" {
			outputData["parent"] = parentIssue
		}
		output, _ := json.MarshalIndent(outputData, "", "  ")
		fmt.Println(string(output))
	} else {
		if parentIssue != "" {
			fmt.Printf("✓ Created subtask: %s (under parent %s)\n", result.Key, parentIssue)
		} else {
			fmt.Printf("✓ Created issue: %s\n", result.Key)
		}
		fmt.Printf("  URL: %s\n", result.Self)
	}

	return nil
}

// loadTemplateData loads template data from a file or stdin
func loadTemplateData(path string) (map[string]interface{}, error) {
	var reader io.Reader

	if path == "-" {
		// Read from stdin
		reader = os.Stdin
	} else {
		// Read from file
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open data file: %w", err)
		}
		defer file.Close()
		reader = file
	}

	// Parse JSON
	var data map[string]interface{}
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON data: %w", err)
	}

	return data, nil
}

// promptForTemplateData prompts the user interactively for template data
// This is a simplified implementation - can be enhanced with better UX
func promptForTemplateData(tmpl *template.Template) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	fmt.Printf("Creating %s issue. Enter field values:\n\n", tmpl.Type)

	// For now, just prompt for common fields
	// This can be enhanced to parse the template and prompt for all placeholders
	commonFields := []string{"Project", "Summary", "Description"}

	for _, field := range commonFields {
		fmt.Printf("%s: ", field)
		var value string
		if _, err := fmt.Scanln(&value); err != nil && err != io.EOF {
			return nil, err
		}
		if value != "" {
			data[field] = value
		}
	}

	return data, nil
}

// ensureRequiredFields ensures that required fields are present
func ensureRequiredFields(fields map[string]interface{}, tmpl *template.Template) error {
	// Check for project
	if fields["project"] == nil {
		return fmt.Errorf("field 'project' is required")
	}

	// Check for issuetype (should be set by template)
	if fields["issuetype"] == nil {
		// Set from template type
		fields["issuetype"] = map[string]interface{}{
			"name": tmpl.Type,
		}
	}

	// Check for summary
	if fields["summary"] == nil {
		return fmt.Errorf("field 'summary' is required")
	}

	return nil
}

// printFields prints fields in a human-readable format
func printFields(fields map[string]interface{}) {
	for key, value := range fields {
		fmt.Printf("  %s: %v\n", key, value)
	}
}

// setupParentIssue validates the parent issue and sets up the parent field for subtask creation.
// It performs the following validations:
//   - Parent issue exists
//   - Parent issue is not itself a subtask (cannot create sub-subtasks)
//
// Then sets the "parent" field in the issue fields map.
func setupParentIssue(fields map[string]interface{}, parentKey string, issueService *jira.IssueService) error {
	if verbose {
		fmt.Printf("Validating parent issue %s...\n", parentKey)
	}

	// Fetch the parent issue to validate it exists
	parentIssue, err := issueService.GetIssue(parentKey)
	if err != nil {
		return fmt.Errorf("parent issue %s not found: %w", parentKey, err)
	}

	// Check if parent is already a subtask (cannot create sub-subtasks)
	if parentIssue.Fields != nil {
		if issueTypeMap, ok := parentIssue.Fields["issuetype"].(map[string]interface{}); ok {
			if subtask, ok := issueTypeMap["subtask"].(bool); ok && subtask {
				return fmt.Errorf("cannot create subtask under %s because it is already a subtask", parentKey)
			}
		}
	}

	// Check the issue type being created - warn if it doesn't look like a subtask type
	if issueType, ok := fields["issuetype"].(map[string]interface{}); ok {
		if name, ok := issueType["name"].(string); ok {
			lowerName := strings.ToLower(name)
			if !strings.Contains(lowerName, "sub") && !strings.Contains(lowerName, "task") {
				if verbose {
					fmt.Printf("Warning: Issue type '%s' may not be a valid subtask type. "+
						"Consider using 'Sub-task' or similar.\n", name)
				}
			}
		}
	}

	// Set the parent field for the Jira API
	// The parent field expects an object with "key" property
	fields["parent"] = map[string]interface{}{
		"key": parentKey,
	}

	if verbose {
		fmt.Printf("✓ Parent issue %s validated. Creating as subtask.\n", parentKey)
	}

	return nil
}
