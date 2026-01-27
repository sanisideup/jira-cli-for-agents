package cmd

import (
	"fmt"
	"strings"

	"github.com/sanisideup/jira-cli-for-agents/pkg/jira"
	"github.com/spf13/cobra"
)

var (
	listProject  string
	listAssignee string
	listStatus   string
	listLimit    int
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues with optional filters",
	Long: `List Jira issues with optional filtering by project, assignee, and status.
By default, lists recent issues for the current user.

Examples:
  jcfa list
  jcfa list --project PROJ
  jcfa list --assignee john@example.com --status "In Progress"
  jcfa list --limit 10 --json`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&listProject, "project", "p", "", "filter by project key")
	listCmd.Flags().StringVarP(&listAssignee, "assignee", "a", "", "filter by assignee email (use 'currentUser()' for yourself)")
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "filter by status")
	listCmd.Flags().IntVarP(&listLimit, "limit", "l", 25, "maximum number of results to return")
}

func runList(cmd *cobra.Command, args []string) error {
	// Build JQL query from flags
	jql := buildJQL()

	if verbose {
		fmt.Printf("JQL: %s\n", jql)
	}

	// Create search service
	searchService := jira.NewSearchService(jiraClient)

	// Execute search
	result, err := searchService.Search(jql, listLimit, nil)
	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	// Use the same output function as search command
	if jsonOutput {
		return outputJSON(result)
	}

	return outputSearchResults(result)
}

// buildJQL builds a JQL query from the list command flags
func buildJQL() string {
	var conditions []string

	// Project filter
	if listProject != "" {
		conditions = append(conditions, fmt.Sprintf("project = %s", listProject))
	}

	// Assignee filter
	if listAssignee != "" {
		// Check if it's currentUser() or a specific email
		if listAssignee == "currentUser()" || listAssignee == "me" {
			conditions = append(conditions, "assignee = currentUser()")
		} else {
			conditions = append(conditions, fmt.Sprintf("assignee = \"%s\"", listAssignee))
		}
	} else {
		// Default to current user if no assignee specified
		conditions = append(conditions, "assignee = currentUser()")
	}

	// Status filter
	if listStatus != "" {
		conditions = append(conditions, fmt.Sprintf("status = \"%s\"", listStatus))
	}

	// Combine conditions
	jql := strings.Join(conditions, " AND ")

	// Add default ordering by updated date
	jql += " ORDER BY updated DESC"

	return jql
}
