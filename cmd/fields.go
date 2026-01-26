package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/sanisideup/jira-cli/pkg/config"
	"github.com/sanisideup/jira-cli/pkg/jira"
	"github.com/sanisideup/jira-cli/pkg/models"
	"github.com/spf13/cobra"
)

var (
	// Flags for fields list command
	projectKey string
)

// fieldsCmd represents the fields command
var fieldsCmd = &cobra.Command{
	Use:   "fields",
	Short: "Manage Jira fields and field mappings",
	Long: `Discover and manage Jira fields, including custom fields.
Use this command to list available fields and create aliases for custom fields.`,
}

// fieldsListCmd represents the fields list command
var fieldsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Jira fields",
	Long: `List all available Jira fields including standard and custom fields.
Output can be filtered by project and formatted as JSON or human-readable table.`,
	Example: `  # List all fields
  jira-cli fields list

  # List fields for a specific project
  jira-cli fields list --project PROJ

  # Output as JSON
  jira-cli fields list --json`,
	RunE: runFieldsList,
}

// fieldsMapCmd represents the fields map command
var fieldsMapCmd = &cobra.Command{
	Use:   "map <alias> <field-id>",
	Short: "Create an alias for a custom field",
	Long: `Map a custom field ID to a human-readable alias.
This makes it easier to reference custom fields in other commands.

The field ID must exist in your Jira instance. Use 'jira-cli fields list' to find field IDs.`,
	Example: `  # Map story_points to customfield_10016
  jira-cli fields map story_points customfield_10016

  # Map epic_link to customfield_10014
  jira-cli fields map epic_link customfield_10014`,
	Args: cobra.ExactArgs(2),
	RunE: runFieldsMap,
}

func init() {
	rootCmd.AddCommand(fieldsCmd)
	fieldsCmd.AddCommand(fieldsListCmd)
	fieldsCmd.AddCommand(fieldsMapCmd)

	// Flags for fields list
	fieldsListCmd.Flags().StringVarP(&projectKey, "project", "p", "", "filter fields by project key")
}

// runFieldsList handles the fields list command
func runFieldsList(cmd *cobra.Command, args []string) error {
	// Create field service
	fieldService := jira.NewFieldService(jiraClient)

	// Fetch fields
	fields, err := fieldService.ListFields(projectKey)
	if err != nil {
		return fmt.Errorf("failed to list fields: %w", err)
	}

	// Output as JSON if requested
	if jsonOutput {
		return outputFieldsJSON(fields)
	}

	// Output as human-readable table
	return outputFieldsTable(fields)
}

// runFieldsMap handles the fields map command
func runFieldsMap(cmd *cobra.Command, args []string) error {
	alias := args[0]
	fieldID := args[1]

	// Validate alias format (no spaces, alphanumeric + underscore)
	if !isValidAlias(alias) {
		return fmt.Errorf("invalid alias '%s': alias must contain only letters, numbers, and underscores", alias)
	}

	// Create field service
	fieldService := jira.NewFieldService(jiraClient)

	// Save field mapping
	if err := fieldService.SaveFieldMapping(alias, fieldID, cfg); err != nil {
		return err
	}

	// Get field details for confirmation
	field, err := fieldService.GetFieldByID(fieldID)
	if err != nil {
		// Mapping was saved but we couldn't get field details
		fmt.Printf("✓ Mapped '%s' to '%s'\n", alias, fieldID)
		return nil
	}

	fmt.Printf("✓ Successfully mapped alias '%s' to field '%s' (%s)\n", alias, field.Name, fieldID)

	configPath, _ := config.GetConfigPath()
	if configPath != "" {
		fmt.Printf("  Configuration saved to: %s\n", configPath)
	}

	return nil
}

// outputFieldsJSON outputs fields in JSON format
func outputFieldsJSON(fields []models.Field) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(fields)
}

// outputFieldsTable outputs fields in a human-readable table format
func outputFieldsTable(fields []models.Field) error {
	// Separate standard and custom fields
	var standardFields []models.Field
	var customFields []models.Field

	for _, field := range fields {
		if field.Custom {
			customFields = append(customFields, field)
		} else {
			standardFields = append(standardFields, field)
		}
	}

	// Sort fields by name
	sort.Slice(standardFields, func(i, j int) bool {
		return standardFields[i].Name < standardFields[j].Name
	})
	sort.Slice(customFields, func(i, j int) bool {
		return customFields[i].Name < customFields[j].Name
	})

	// Create tab writer for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print standard fields
	if len(standardFields) > 0 {
		fmt.Fprintln(w, "Standard Fields:")
		fmt.Fprintln(w, "ID\tNAME\tTYPE")
		fmt.Fprintln(w, strings.Repeat("-", 80))

		for _, field := range standardFields {
			fieldType := field.Schema.Type
			if field.Schema.System != "" {
				fieldType = field.Schema.System
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", field.ID, field.Name, fieldType)
		}
		w.Flush()
		fmt.Println()
	}

	// Print custom fields
	if len(customFields) > 0 {
		fmt.Fprintln(w, "Custom Fields:")
		fmt.Fprintln(w, "ID\tNAME\tTYPE")
		fmt.Fprintln(w, strings.Repeat("-", 80))

		for _, field := range customFields {
			fieldType := field.Schema.Type
			if field.Schema.Custom != "" {
				// Extract the last part of custom type for readability
				parts := strings.Split(field.Schema.Custom, ":")
				if len(parts) > 0 {
					fieldType = parts[len(parts)-1]
				}
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", field.ID, field.Name, fieldType)
		}
		w.Flush()
	}

	// Print summary
	fmt.Printf("\nTotal: %d fields (%d standard, %d custom)\n",
		len(fields), len(standardFields), len(customFields))

	// Print current mappings if any exist
	if cfg != nil && len(cfg.FieldMappings) > 0 {
		fmt.Println("\nCurrent Field Mappings:")

		// Sort aliases for consistent output
		var aliases []string
		for alias := range cfg.FieldMappings {
			aliases = append(aliases, alias)
		}
		sort.Strings(aliases)

		fmt.Fprintln(w, "ALIAS\tFIELD ID")
		fmt.Fprintln(w, strings.Repeat("-", 40))
		for _, alias := range aliases {
			fmt.Fprintf(w, "%s\t%s\n", alias, cfg.FieldMappings[alias])
		}
		w.Flush()
	}

	return nil
}

// isValidAlias checks if an alias contains only valid characters
func isValidAlias(alias string) bool {
	if alias == "" {
		return false
	}

	for _, ch := range alias {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '_') {
			return false
		}
	}

	return true
}
