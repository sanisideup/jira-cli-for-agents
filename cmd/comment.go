package cmd

import (
	"fmt"

	"github.com/sanisideup/jira-cli/pkg/jira"
	"github.com/spf13/cobra"
)

// commentCmd maintains backward compatibility
// Old syntax: jira-cli comment PROJ-123 "text"
// Routes to the new comments add subcommand
var commentCmd = &cobra.Command{
	Use:   "comment <issue-key> \"<text>\"",
	Short: "Add a comment to a Jira issue (legacy command)",
	Long: `Add a text comment to an existing Jira issue.

This is a legacy command maintained for backward compatibility.
Use 'jira-cli comments' for more advanced comment operations.

Examples:
  jira-cli comment PROJ-123 "This is a comment"
  jira-cli comment PROJ-123 "Updated the implementation" --json

For more options, see:
  jira-cli comments --help`,
	Args: cobra.ExactArgs(2),
	RunE: runComment,
}

func runComment(cmd *cobra.Command, args []string) error {
	issueKey := args[0]
	commentText := args[1]

	if verbose {
		fmt.Printf("Adding comment to issue %s\n", issueKey)
	}

	// Use the new CommentService
	commentService := jira.NewCommentService(jiraClient)

	// Add the comment
	comment, err := commentService.AddComment(issueKey, commentText)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	if jsonOutput {
		return outputJSON(comment)
	}

	fmt.Printf("✓ Successfully added comment to issue %s\n", issueKey)
	if verbose && comment != nil {
		fmt.Printf("Comment ID: %s\n", comment.ID)
	}

	return nil
}
