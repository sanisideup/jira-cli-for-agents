package cmd

import (
	"fmt"

	"github.com/sanisideup/jira-cli-for-agents/pkg/jira"
	"github.com/spf13/cobra"
)

var transitionCmd = &cobra.Command{
	Use:   "transition <issue-key> \"<status>\"",
	Short: "Transition an issue to a new status",
	Long: `Transition a Jira issue to a new workflow status.

The status name is case-insensitive. If the specified status is not available
for the issue, the command will show available transitions.

Examples:
  jcfa transition PROJ-123 "In Progress"
  jcfa transition PROJ-123 "Done" --json`,
	Args: cobra.ExactArgs(2),
	RunE: runTransition,
}

func init() {
	rootCmd.AddCommand(transitionCmd)
}

func runTransition(cmd *cobra.Command, args []string) error {
	issueKey := args[0]
	targetStatus := args[1]

	if verbose {
		fmt.Printf("Transitioning issue %s to '%s'\n", issueKey, targetStatus)
	}

	// Create search service
	searchService := jira.NewSearchService(jiraClient)

	// Transition the issue
	if err := searchService.TransitionIssue(issueKey, targetStatus); err != nil {
		return fmt.Errorf("failed to transition issue: %w", err)
	}

	if jsonOutput {
		return outputJSON(map[string]interface{}{
			"status":  "success",
			"message": fmt.Sprintf("Successfully transitioned issue %s to '%s'", issueKey, targetStatus),
		})
	}

	fmt.Printf("✓ Successfully transitioned issue %s to '%s'\n", issueKey, targetStatus)
	return nil
}
