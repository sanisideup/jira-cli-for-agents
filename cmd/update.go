package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sanisideup/jira-cli/pkg/jira"
	"github.com/spf13/cobra"
)

var (
	updateFields []string
)

var updateCmd = &cobra.Command{
	Use:   "update <issue-key>",
	Short: "Update fields on a Jira issue",
	Long: `Update one or more fields on an existing Jira issue.

Field values can be specified using the --field flag multiple times.
Field names can be either field IDs (like "customfield_10016") or aliases
configured in your field mappings.

Examples:
  jira-cli update PROJ-123 --field summary="New title"
  jira-cli update PROJ-123 --field story_points=8
  jira-cli update PROJ-123 --field summary="Updated" --field description="New desc"`,
	Args: cobra.ExactArgs(1),
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().StringArrayVarP(&updateFields, "field", "f", []string{}, "field to update in format name=value (can be specified multiple times)")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	issueKey := args[0]

	if len(updateFields) == 0 {
		return fmt.Errorf("at least one field must be specified using --field")
	}

	// Parse field values
	fields, err := parseFieldUpdates(updateFields)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("Updating issue %s with fields: %v\n", issueKey, fields)
	}

	// Create search service
	searchService := jira.NewSearchService(jiraClient)

	// Update the issue
	if err := searchService.UpdateIssue(issueKey, fields); err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}

	if jsonOutput {
		return outputJSON(map[string]interface{}{
			"status":  "success",
			"message": fmt.Sprintf("Successfully updated issue %s", issueKey),
			"fields":  fields,
		})
	}

	fmt.Printf("✓ Successfully updated issue %s\n", issueKey)
	return nil
}

// parseFieldUpdates parses field update strings in format "name=value"
func parseFieldUpdates(fieldStrs []string) (map[string]interface{}, error) {
	fields := make(map[string]interface{})

	for _, fieldStr := range fieldStrs {
		// Split on first '=' only
		parts := strings.SplitN(fieldStr, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid field format '%s': expected name=value", fieldStr)
		}

		fieldName := strings.TrimSpace(parts[0])
		fieldValue := strings.TrimSpace(parts[1])

		// Resolve field name to field ID (check mappings)
		fieldID := resolveFieldName(fieldName)

		// Parse value based on field type
		parsedValue := parseFieldValue(fieldID, fieldValue)

		fields[fieldID] = parsedValue
	}

	return fields, nil
}

// resolveFieldName resolves a field name or alias to a field ID
func resolveFieldName(name string) string {
	// Check if it's already a field ID
	if strings.HasPrefix(name, "customfield_") || isStandardField(name) {
		return name
	}

	// Check field mappings in config
	if cfg != nil && cfg.FieldMappings != nil {
		if fieldID, ok := cfg.FieldMappings[name]; ok {
			return fieldID
		}
	}

	// Return as-is if no mapping found
	return name
}

// isStandardField checks if a field name is a standard Jira field
func isStandardField(name string) bool {
	standardFields := []string{
		"summary", "description", "assignee", "reporter", "priority",
		"labels", "status", "issuetype", "project", "components",
		"fixVersions", "affectedVersions", "duedate", "parent",
	}

	for _, field := range standardFields {
		if name == field {
			return true
		}
	}

	return false
}

// parseFieldValue parses a field value string into the appropriate type
func parseFieldValue(fieldID, value string) interface{} {
	// Handle special fields that need object format
	switch fieldID {
	case "assignee":
		if value == "null" || value == "" {
			return nil
		}
		return map[string]interface{}{"accountId": value}

	case "priority":
		return map[string]interface{}{"name": value}

	case "issuetype":
		return map[string]interface{}{"name": value}

	case "project":
		return map[string]interface{}{"key": value}

	case "labels":
		// Parse as comma-separated list
		if value == "" {
			return []string{}
		}
		return strings.Split(value, ",")
	}

	// Try to parse as number for custom fields
	if strings.HasPrefix(fieldID, "customfield_") {
		if num, err := strconv.ParseFloat(value, 64); err == nil {
			return num
		}
	}

	// Return as string for other fields
	return value
}
