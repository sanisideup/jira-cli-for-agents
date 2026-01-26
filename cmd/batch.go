package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sanisideup/jira-cli/pkg/jira"
	"github.com/sanisideup/jira-cli/pkg/models"
	"github.com/sanisideup/jira-cli/pkg/template"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	batchFile   string
	noProgress  bool
	batchDryRun bool
)

// BatchItem represents a single item in the batch input
type BatchItem struct {
	Template string                 `json:"template"`
	Data     map[string]interface{} `json:"data"`
	ID       string                 `json:"id,omitempty"` // Optional ID for referencing
}

// BatchResult represents the result of batch creation
type BatchResult struct {
	Success int                       `json:"success"`
	Failed  int                       `json:"failed"`
	Created []CreatedIssue            `json:"created"`
	Errors  []BatchError              `json:"errors"`
}

// CreatedIssue represents a successfully created issue
type CreatedIssue struct {
	Key     string `json:"key"`
	Type    string `json:"type"`
	Summary string `json:"summary"`
}

// BatchError represents an error during batch creation
type BatchError struct {
	Index int                    `json:"index"`
	Error string                 `json:"error"`
	Data  map[string]interface{} `json:"data"`
}

// batchCmd represents the batch create command
var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Batch operations on Jira issues",
	Long:  `Perform batch operations on multiple Jira issues.`,
}

// batchCreateCmd represents the batch create command
var batchCreateCmd = &cobra.Command{
	Use:   "create <json-file>",
	Short: "Create multiple Jira issues from a JSON file",
	Long: `Create multiple Jira issues from a JSON file.

The input JSON file should contain an array of objects with the following structure:
[
  {
    "template": "epic",
    "id": "epic1",
    "data": {
      "Project": "PROJ",
      "Summary": "Q1 Platform Epic",
      "Description": "Epic description",
      "EpicName": "Q1 Platform"
    }
  },
  {
    "template": "story",
    "data": {
      "Project": "PROJ",
      "Summary": "User authentication",
      "Description": "Story description",
      "EpicKey": "@epic1",
      "StoryPoints": 5
    }
  }
]

Use @<id> to reference other issues in the batch (e.g., "@epic1" links to the epic created with id "epic1").

Examples:
  # Create issues from a file
  jira-cli batch create issues.json

  # Dry run to validate
  jira-cli batch create issues.json --dry-run

  # Disable progress bar
  jira-cli batch create issues.json --no-progress

  # JSON output
  jira-cli batch create issues.json --json
`,
	Args: cobra.ExactArgs(1),
	RunE: runBatchCreate,
}

func init() {
	rootCmd.AddCommand(batchCmd)
	batchCmd.AddCommand(batchCreateCmd)

	batchCreateCmd.Flags().BoolVar(&batchDryRun, "dry-run", false, "validate without creating issues")
	batchCreateCmd.Flags().BoolVar(&noProgress, "no-progress", false, "disable progress bar")
}

func runBatchCreate(cmd *cobra.Command, args []string) error {
	batchFile = args[0]

	// Load batch items
	items, err := loadBatchItems(batchFile)
	if err != nil {
		return fmt.Errorf("failed to load batch file: %w", err)
	}

	if len(items) == 0 {
		return fmt.Errorf("batch file contains no items")
	}

	// Initialize services
	templateService := template.NewService(filepath.Join(os.Getenv("HOME"), ".jira-cli", "templates"))
	issueService := jira.NewIssueService(jiraClient)
	linkService := jira.NewLinkService(jiraClient)

	// Prepare all issues
	preparedItems, err := prepareBatchItems(items, templateService)
	if err != nil {
		return fmt.Errorf("failed to prepare batch items: %w", err)
	}

	// Validate all issues
	for i, item := range preparedItems {
		if err := issueService.ValidateIssueFields(item.Fields); err != nil {
			return fmt.Errorf("validation failed for item %d: %w", i, err)
		}
	}

	// Dry run: just show what would be created
	if batchDryRun {
		if jsonOutput {
			output, _ := json.MarshalIndent(preparedItems, "", "  ")
			fmt.Println(string(output))
		} else {
			fmt.Printf("✓ Validation passed. Would create %d issues:\n", len(preparedItems))
			for i, item := range preparedItems {
				summary := item.Fields["summary"]
				fmt.Printf("  %d. %s: %v\n", i+1, item.Type, summary)
			}
		}
		return nil
	}

	// Separate epics from other issues
	epics, others := separateEpics(preparedItems)

	// Create issues
	result := &BatchResult{
		Created: make([]CreatedIssue, 0),
		Errors:  make([]BatchError, 0),
	}

	// ID to issue key mapping for referencing
	idToKey := make(map[string]string)

	// Progress bar
	var bar *progressbar.ProgressBar
	if !noProgress && !jsonOutput {
		bar = progressbar.NewOptions(len(preparedItems),
			progressbar.OptionSetDescription("Creating issues..."),
			progressbar.OptionSetWidth(15),
			progressbar.OptionShowCount(),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "█",
				SaucerPadding: "░",
				BarStart:      "[",
				BarEnd:        "]",
			}),
		)
	}

	// Create epics first
	if len(epics) > 0 {
		createBatch(epics, issueService, result, idToKey, bar)
	}

	// Update references in other issues (e.g., @epic1 -> PROJ-123)
	resolveReferences(others, idToKey)

	// Create other issues
	if len(others) > 0 {
		createBatch(others, issueService, result, idToKey, bar)
	}

	// Link stories to epics if needed
	linkStoriesToEpics(others, linkService, result)

	// Calculate totals
	result.Success = len(result.Created)
	result.Failed = len(result.Errors)

	// Output results
	if jsonOutput {
		output, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(output))
	} else {
		if bar != nil {
			fmt.Println() // New line after progress bar
		}
		printBatchResult(result)
	}

	// Exit with error code if there were failures
	if result.Failed > 0 {
		os.Exit(2) // Validation/creation error exit code
	}

	return nil
}

// PreparedItem represents an item ready for creation
type PreparedItem struct {
	Index    int
	ID       string
	Type     string
	Fields   map[string]interface{}
	Template string
	RawData  map[string]interface{}
}

// loadBatchItems loads batch items from a JSON file
func loadBatchItems(path string) ([]BatchItem, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var items []BatchItem
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return items, nil
}

// prepareBatchItems prepares all batch items by rendering templates
func prepareBatchItems(items []BatchItem, templateService *template.Service) ([]PreparedItem, error) {
	prepared := make([]PreparedItem, 0, len(items))

	for i, item := range items {
		// Load template
		tmpl, err := templateService.LoadTemplate(item.Template)
		if err != nil {
			return nil, fmt.Errorf("item %d: failed to load template '%s': %w", i, item.Template, err)
		}

		// Render template
		fields, err := templateService.RenderTemplate(tmpl, item.Data, cfg)
		if err != nil {
			return nil, fmt.Errorf("item %d: failed to render template: %w", i, err)
		}

		// Ensure required fields
		if fields["issuetype"] == nil {
			fields["issuetype"] = map[string]interface{}{"name": tmpl.Type}
		}

		prepared = append(prepared, PreparedItem{
			Index:    i,
			ID:       item.ID,
			Type:     tmpl.Type,
			Fields:   fields,
			Template: item.Template,
			RawData:  item.Data,
		})
	}

	return prepared, nil
}

// separateEpics separates epics from other issue types
func separateEpics(items []PreparedItem) (epics []PreparedItem, others []PreparedItem) {
	for _, item := range items {
		if strings.ToLower(item.Type) == "epic" {
			epics = append(epics, item)
		} else {
			others = append(others, item)
		}
	}
	return
}

// createBatch creates a batch of issues
func createBatch(items []PreparedItem, service *jira.IssueService, result *BatchResult, idToKey map[string]string, bar *progressbar.ProgressBar) {
	// Extract fields for bulk creation
	fieldsArray := make([]map[string]interface{}, len(items))
	for i, item := range items {
		fieldsArray[i] = item.Fields
	}

	// Create issues
	response, err := service.BulkCreateIssues(fieldsArray)
	if err != nil {
		// If bulk creation fails entirely, record as errors
		for _, item := range items {
			result.Errors = append(result.Errors, BatchError{
				Index: item.Index,
				Error: err.Error(),
				Data:  item.RawData,
			})
			if bar != nil {
				bar.Add(1)
			}
		}
		return
	}

	// Process successful creations
	for i, created := range response.Issues {
		if i < len(items) {
			item := items[i]

			// Store ID mapping
			if item.ID != "" {
				idToKey[item.ID] = created.Key
			}

			// Add to results
			summary := ""
			if s, ok := item.Fields["summary"].(string); ok {
				summary = s
			}

			result.Created = append(result.Created, CreatedIssue{
				Key:     created.Key,
				Type:    item.Type,
				Summary: summary,
			})

			if bar != nil {
				bar.Add(1)
			}
		}
	}

	// Process errors
	for _, bulkErr := range response.Errors {
		idx := bulkErr.FailedElementNumber
		if idx < len(items) {
			item := items[idx]
			errorMsg := formatBulkError(&bulkErr.ElementErrors)

			result.Errors = append(result.Errors, BatchError{
				Index: item.Index,
				Error: errorMsg,
				Data:  item.RawData,
			})

			if bar != nil {
				bar.Add(1)
			}
		}
	}
}

// resolveReferences resolves @id references to actual issue keys
func resolveReferences(items []PreparedItem, idToKey map[string]string) {
	for i := range items {
		resolveFieldReferences(items[i].Fields, idToKey)
	}
}

// resolveFieldReferences recursively resolves @id references in fields
func resolveFieldReferences(fields map[string]interface{}, idToKey map[string]string) {
	for key, value := range fields {
		switch v := value.(type) {
		case string:
			// Check if it's a reference (starts with @)
			if strings.HasPrefix(v, "@") {
				refID := strings.TrimPrefix(v, "@")
				if issueKey, exists := idToKey[refID]; exists {
					fields[key] = issueKey
				}
			}
		case map[string]interface{}:
			// Recursively resolve nested objects
			resolveFieldReferences(v, idToKey)
		case []interface{}:
			// Recursively resolve arrays
			for j, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					resolveFieldReferences(m, idToKey)
				} else if s, ok := item.(string); ok && strings.HasPrefix(s, "@") {
					refID := strings.TrimPrefix(s, "@")
					if issueKey, exists := idToKey[refID]; exists {
						v[j] = issueKey
					}
				}
			}
		}
	}
}

// linkStoriesToEpics links stories to epics based on EpicKey field
func linkStoriesToEpics(items []PreparedItem, linkService *jira.LinkService, result *BatchResult) {
	// This is a best-effort operation - we don't fail the batch if linking fails
	for _, item := range items {
		// Look for epic link in fields
		epicKey := findEpicKey(item.Fields)
		if epicKey == "" {
			continue
		}

		// Find the story key from created issues
		storyKey := ""
		for _, created := range result.Created {
			if created.Type == item.Type {
				storyKey = created.Key
				break
			}
		}

		if storyKey == "" {
			continue
		}

		// Link to epic (ignore errors)
		_ = linkService.LinkToEpic(storyKey, epicKey, cfg)
	}
}

// findEpicKey finds the epic key in fields (checks common field names)
func findEpicKey(fields map[string]interface{}) string {
	// Check common epic link field names/aliases
	epicFields := []string{"epic", "epicKey", "EpicKey", "epic_link", "customfield_10014"}

	for _, fieldName := range epicFields {
		if value, exists := fields[fieldName]; exists {
			if strValue, ok := value.(string); ok {
				return strValue
			}
		}
	}

	return ""
}

// formatBulkError formats a bulk error response
func formatBulkError(errResp *models.ErrorResponse) string {
	var messages []string

	if len(errResp.ErrorMessages) > 0 {
		messages = append(messages, strings.Join(errResp.ErrorMessages, "; "))
	}

	if len(errResp.Errors) > 0 {
		for field, msg := range errResp.Errors {
			messages = append(messages, fmt.Sprintf("%s: %s", field, msg))
		}
	}

	if len(messages) == 0 {
		return "unknown error"
	}

	return strings.Join(messages, "; ")
}

// printBatchResult prints the batch result in a human-readable format
func printBatchResult(result *BatchResult) {
	if result.Success > 0 {
		fmt.Printf("✓ Successfully created %d issue(s):\n", result.Success)
		for _, created := range result.Created {
			fmt.Printf("  %s: %s - %s\n", created.Key, created.Type, created.Summary)
		}
		fmt.Println()
	}

	if result.Failed > 0 {
		fmt.Printf("✗ Failed to create %d issue(s):\n", result.Failed)
		for _, err := range result.Errors {
			fmt.Printf("  Item #%d: %s\n", err.Index+1, err.Error)
			if verbose {
				dataJSON, _ := json.MarshalIndent(err.Data, "    ", "  ")
				fmt.Printf("    Data: %s\n", string(dataJSON))
			}
		}
	}
}
