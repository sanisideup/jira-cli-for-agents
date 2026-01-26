package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sanisideup/jira-cli/pkg/jira"
	"github.com/spf13/cobra"
)

var (
	searchLimit int
)

var searchCmd = &cobra.Command{
	Use:   "search \"<JQL>\"",
	Short: "Search for issues using JQL",
	Long: `Search for Jira issues using Jira Query Language (JQL).

Examples:
  jira-cli search "project = PROJ AND status = Open"
  jira-cli search "assignee = currentUser() ORDER BY updated DESC" --limit 20
  jira-cli search "project = PROJ AND type = Story" --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().IntVar(&searchLimit, "limit", 50, "maximum number of results to return")
}

func runSearch(cmd *cobra.Command, args []string) error {
	jql := args[0]

	// Create search service
	searchService := jira.NewSearchService(jiraClient)

	// Execute search
	result, err := searchService.Search(jql, searchLimit, nil)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Output based on format
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
		return nil
	}

	return outputSearchResults(result)
}

// outputSearchResults outputs search results in human-readable format
func outputSearchResults(result interface{}) error {
	// Type assert or convert to map
	var data map[string]interface{}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return fmt.Errorf("failed to unmarshal results: %w", err)
	}

	// Extract issues
	issues, ok := data["issues"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid search result format")
	}

	total, _ := data["total"].(float64)

	if len(issues) == 0 {
		fmt.Println("No issues found")
		return nil
	}

	fmt.Printf("Found %d issues:\n", int(total))
	fmt.Println()

	// Print table header
	fmt.Printf("%-12s %-12s %-15s %s\n", "KEY", "TYPE", "STATUS", "SUMMARY")
	fmt.Println("--------------------------------------------------------------------------------")

	// Print each issue
	for _, issueItem := range issues {
		issue, ok := issueItem.(map[string]interface{})
		if !ok {
			continue
		}

		key, _ := issue["key"].(string)
		fields, ok := issue["fields"].(map[string]interface{})
		if !ok {
			continue
		}

		// Extract fields
		summary, _ := fields["summary"].(string)

		var issueType string
		if it, ok := fields["issuetype"].(map[string]interface{}); ok {
			issueType, _ = it["name"].(string)
		}

		var status string
		if st, ok := fields["status"].(map[string]interface{}); ok {
			status, _ = st["name"].(string)
		}

		// Truncate summary if too long
		if len(summary) > 50 {
			summary = summary[:47] + "..."
		}

		// Print row
		fmt.Printf("%-12s %-12s %-15s %s\n",
			truncate(key, 12),
			truncate(issueType, 12),
			truncate(status, 15),
			summary,
		)
	}

	return nil
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
