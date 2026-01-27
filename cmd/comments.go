package cmd

import (
	"fmt"
	"strings"

	"github.com/sanisideup/jira-cli-for-agents/pkg/jira"
	"github.com/spf13/cobra"
)

var (
	commentLimit   int
	commentOrder   string
	commentConfirm bool
)

// commentsCmd is the new parent command for comment operations
var commentsCmd = &cobra.Command{
	Use:   "comments <subcommand>",
	Short: "Manage comments on Jira issues",
	Long: `Manage comments on Jira issues with various operations.

Subcommands:
  add     - Add a comment to an issue
  list    - List all comments on an issue
  get     - Get a specific comment
  update  - Update an existing comment
  delete  - Delete a comment

For backward compatibility, you can still use:
  jcfa comment PROJ-123 "text"`,
	Run: func(cmd *cobra.Command, args []string) {
		// If called without subcommand, show help
		cmd.Help()
	},
}

// commentAddCmd adds a comment to an issue
var commentAddCmd = &cobra.Command{
	Use:   "add <issue-key> \"<text>\"",
	Short: "Add a comment to an issue",
	Long: `Add a text comment to an existing Jira issue.

Examples:
  jcfa comments add PROJ-123 "This is a comment"
  jcfa comments add PROJ-123 "Updated the implementation" --json`,
	Args: cobra.ExactArgs(2),
	RunE: runCommentAdd,
}

// commentListCmd lists all comments on an issue
var commentListCmd = &cobra.Command{
	Use:   "list <issue-key>",
	Short: "List all comments on an issue",
	Long: `List all comments on an issue with pagination support.

Examples:
  jcfa comments list PROJ-123
  jcfa comments list PROJ-123 --limit 10
  jcfa comments list PROJ-123 --order -created --json`,
	Args: cobra.ExactArgs(1),
	RunE: runCommentList,
}

// commentGetCmd gets a specific comment
var commentGetCmd = &cobra.Command{
	Use:   "get <issue-key> <comment-id>",
	Short: "Get a specific comment",
	Long: `Get the full details of a specific comment by ID.

Examples:
  jcfa comments get PROJ-123 10001
  jcfa comments get PROJ-123 10001 --json`,
	Args: cobra.ExactArgs(2),
	RunE: runCommentGet,
}

// commentUpdateCmd updates an existing comment
var commentUpdateCmd = &cobra.Command{
	Use:   "update <issue-key> <comment-id> \"<new-text>\"",
	Short: "Update an existing comment",
	Long: `Update the text of an existing comment.

Note: You can only update comments you created or if you have admin permissions.

Examples:
  jcfa comments update PROJ-123 10001 "Updated comment text"
  jcfa comments update PROJ-123 10001 "Fixed typo" --json`,
	Args: cobra.ExactArgs(3),
	RunE: runCommentUpdate,
}

// commentDeleteCmd deletes a comment
var commentDeleteCmd = &cobra.Command{
	Use:   "delete <issue-key> <comment-id>",
	Short: "Delete a comment",
	Long: `Delete a comment from an issue.

Note: You can only delete comments you created or if you have admin permissions.
Requires --confirm flag for safety.

Examples:
  jcfa comments delete PROJ-123 10001 --confirm
  jcfa comments delete PROJ-123 10002 --confirm --json`,
	Args: cobra.ExactArgs(2),
	RunE: runCommentDelete,
}

func init() {
	// Add subcommands to comments
	commentsCmd.AddCommand(commentAddCmd)
	commentsCmd.AddCommand(commentListCmd)
	commentsCmd.AddCommand(commentGetCmd)
	commentsCmd.AddCommand(commentUpdateCmd)
	commentsCmd.AddCommand(commentDeleteCmd)

	// Add flags
	commentListCmd.Flags().IntVar(&commentLimit, "limit", 0, "Limit number of comments (0 = all)")
	commentListCmd.Flags().StringVar(&commentOrder, "order", "created", "Sort order (created or -created)")

	commentDeleteCmd.Flags().BoolVar(&commentConfirm, "confirm", false, "Confirm deletion")
	commentDeleteCmd.MarkFlagRequired("confirm")

	// Register comments command
	rootCmd.AddCommand(commentsCmd)

	// Keep old comment command for backward compatibility
	// It will intelligently route to the add subcommand
	rootCmd.AddCommand(commentCmd)
}

func runCommentAdd(cmd *cobra.Command, args []string) error {
	issueKey := args[0]
	commentText := args[1]

	if verbose {
		fmt.Printf("Adding comment to issue %s\n", issueKey)
	}

	// Create comment service
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

func runCommentList(cmd *cobra.Command, args []string) error {
	issueKey := args[0]

	if verbose {
		fmt.Printf("Listing comments for issue %s\n", issueKey)
	}

	// Create comment service
	commentService := jira.NewCommentService(jiraClient)

	// List comments
	result, err := commentService.ListComments(issueKey, commentOrder)
	if err != nil {
		return fmt.Errorf("failed to list comments: %w", err)
	}

	if jsonOutput {
		return outputJSON(result)
	}

	// Display comments
	if len(result.Comments) == 0 {
		fmt.Printf("No comments found on issue %s\n", issueKey)
		return nil
	}

	fmt.Printf("Comments for %s (%d total):\n\n", issueKey, result.Total)

	// Apply limit if specified
	comments := result.Comments
	if commentLimit > 0 && commentLimit < len(comments) {
		comments = comments[:commentLimit]
	}

	for _, comment := range comments {
		fmt.Printf("ID: %s\n", comment.ID)
		fmt.Printf("Author: %s\n", comment.Author.DisplayName)
		fmt.Printf("Date: %s\n", jira.FormatDate(comment.Created))

		// Extract plain text from ADF
		text := jira.ExtractPlainText(comment.Body)
		fmt.Printf("Text: %s\n", strings.TrimSpace(text))
		fmt.Println()
	}

	return nil
}

func runCommentGet(cmd *cobra.Command, args []string) error {
	issueKey := args[0]
	commentID := args[1]

	if verbose {
		fmt.Printf("Getting comment %s from issue %s\n", commentID, issueKey)
	}

	// Create comment service
	commentService := jira.NewCommentService(jiraClient)

	// Get comment
	comment, err := commentService.GetComment(issueKey, commentID)
	if err != nil {
		return fmt.Errorf("failed to get comment: %w", err)
	}

	if jsonOutput {
		return outputJSON(comment)
	}

	// Display comment
	fmt.Printf("Comment %s on %s\n\n", comment.ID, issueKey)
	fmt.Printf("Author: %s\n", comment.Author.DisplayName)
	fmt.Printf("Created: %s\n", jira.FormatDate(comment.Created))
	if comment.Updated != comment.Created {
		fmt.Printf("Updated: %s\n", jira.FormatDate(comment.Updated))
		fmt.Printf("Updated by: %s\n", comment.UpdateAuthor.DisplayName)
	}
	fmt.Println()

	// Extract plain text from ADF
	text := jira.ExtractPlainText(comment.Body)
	fmt.Printf("Text:\n%s\n", strings.TrimSpace(text))

	return nil
}

func runCommentUpdate(cmd *cobra.Command, args []string) error {
	issueKey := args[0]
	commentID := args[1]
	newText := args[2]

	if verbose {
		fmt.Printf("Updating comment %s on issue %s\n", commentID, issueKey)
	}

	// Create comment service
	commentService := jira.NewCommentService(jiraClient)

	// Update comment
	err := commentService.UpdateComment(issueKey, commentID, newText)
	if err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	if jsonOutput {
		return outputJSON(map[string]string{
			"status":  "success",
			"message": "Comment updated successfully",
		})
	}

	fmt.Printf("✓ Successfully updated comment %s on issue %s\n", commentID, issueKey)

	return nil
}

func runCommentDelete(cmd *cobra.Command, args []string) error {
	issueKey := args[0]
	commentID := args[1]

	if !commentConfirm {
		return fmt.Errorf("deletion requires --confirm flag for safety")
	}

	if verbose {
		fmt.Printf("Deleting comment %s from issue %s\n", commentID, issueKey)
	}

	// Create comment service
	commentService := jira.NewCommentService(jiraClient)

	// Delete comment
	err := commentService.DeleteComment(issueKey, commentID)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	if jsonOutput {
		return outputJSON(map[string]string{
			"status":  "success",
			"message": "Comment deleted successfully",
		})
	}

	fmt.Printf("✓ Successfully deleted comment %s from issue %s\n", commentID, issueKey)

	return nil
}
