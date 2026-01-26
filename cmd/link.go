package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sanisideup/jira-cli/pkg/jira"
	"github.com/spf13/cobra"
)

// -----------------------------------------------------------------------------
// Command-level flags
// -----------------------------------------------------------------------------

var (
	linkType        string // Link type for create operation
	linkConfirm     bool   // Confirmation flag for delete operation
)

// -----------------------------------------------------------------------------
// Issue key validation
// -----------------------------------------------------------------------------

// issueKeyPattern matches standard Jira issue keys (e.g., "PROJ-123")
var issueKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]+-\d+$`)

// looksLikeIssueKey checks if a string looks like a Jira issue key.
// Used for backward compatibility to detect legacy command usage.
func looksLikeIssueKey(s string) bool {
	return issueKeyPattern.MatchString(strings.ToUpper(s))
}

// -----------------------------------------------------------------------------
// Parent command: link
// -----------------------------------------------------------------------------

// linkCmd is the parent command for all link operations
var linkCmd = &cobra.Command{
	Use:   "link <subcommand>",
	Short: "Manage issue links",
	Long: `Manage links between Jira issues with various operations.

Subcommands:
  create  - Create a link between two issues
  types   - List available link types
  list    - List all links on an issue
  delete  - Delete a link by ID

For backward compatibility, you can still use:
  jira-cli link PROJ-123 PROJ-456 --type Blocks`,
	Args: cobra.ArbitraryArgs,
	RunE: runLinkLegacy,
}

// -----------------------------------------------------------------------------
// Subcommand: link create
// -----------------------------------------------------------------------------

// linkCreateCmd creates a link between two issues
var linkCreateCmd = &cobra.Command{
	Use:   "create <issue-key-1> <issue-key-2>",
	Short: "Create a link between two issues",
	Long: `Create a link between two Jira issues.

The first issue is the "inward" issue and the second is the "outward" issue.
For example, with --type Blocks:
  - PROJ-123 "blocks" PROJ-456
  - PROJ-456 "is blocked by" PROJ-123

Common link types:
  - Blocks: The first issue blocks the second
  - Relates: The issues are related
  - Duplicate: The first issue duplicates the second
  - Epic: Link a story to an epic

Examples:
  jira-cli link create PROJ-123 PROJ-456 --type Blocks
  jira-cli link create PROJ-123 PROJ-456 --type Relates
  jira-cli link create PROJ-100 PROJ-101 --type Epic --json`,
	Args: cobra.ExactArgs(2),
	RunE: runLinkCreate,
}

// -----------------------------------------------------------------------------
// Subcommand: link types
// -----------------------------------------------------------------------------

// linkTypesCmd lists all available link types
var linkTypesCmd = &cobra.Command{
	Use:   "types",
	Short: "List available link types",
	Long: `List all available issue link types in your Jira instance.

Each link type has an inward and outward description that describes
the relationship direction between issues.

Examples:
  jira-cli link types
  jira-cli link types --json`,
	Args: cobra.NoArgs,
	RunE: runLinkTypes,
}

// -----------------------------------------------------------------------------
// Subcommand: link list
// -----------------------------------------------------------------------------

// linkListCmd lists all links on an issue
var linkListCmd = &cobra.Command{
	Use:   "list <issue-key>",
	Short: "List all links on an issue",
	Long: `List all links associated with a specific issue.

Shows the link ID (for deletion), direction, link type, and linked issue details.

Examples:
  jira-cli link list PROJ-123
  jira-cli link list PROJ-123 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runLinkList,
}

// -----------------------------------------------------------------------------
// Subcommand: link delete
// -----------------------------------------------------------------------------

// linkDeleteCmd deletes a link by ID
var linkDeleteCmd = &cobra.Command{
	Use:   "delete <link-id>",
	Short: "Delete a link by ID",
	Long: `Delete an issue link by its ID.

You can find link IDs using 'jira-cli link list <issue-key>'.
Requires --confirm flag for safety.

Examples:
  jira-cli link delete 10234 --confirm
  jira-cli link delete 10234 --confirm --json`,
	Args: cobra.ExactArgs(1),
	RunE: runLinkDelete,
}

// -----------------------------------------------------------------------------
// Initialization
// -----------------------------------------------------------------------------

func init() {
	// Add subcommands to link
	linkCmd.AddCommand(linkCreateCmd)
	linkCmd.AddCommand(linkTypesCmd)
	linkCmd.AddCommand(linkListCmd)
	linkCmd.AddCommand(linkDeleteCmd)

	// Create subcommand flags
	linkCreateCmd.Flags().StringVarP(&linkType, "type", "t", "Relates",
		"type of link (Blocks, Relates, Duplicate, Epic, etc.)")

	// Delete subcommand flags
	linkDeleteCmd.Flags().BoolVar(&linkConfirm, "confirm", false,
		"confirm deletion (required for safety)")
	linkDeleteCmd.MarkFlagRequired("confirm")

	// Legacy support: add --type flag to parent command for backward compatibility.
	// This binds to the same linkType variable as linkCreateCmd, allowing:
	//   jira-cli link PROJ-1 PROJ-2 --type Blocks  (legacy, handled by runLinkLegacy)
	//   jira-cli link create PROJ-1 PROJ-2 --type Blocks  (new style)
	linkCmd.Flags().StringVarP(&linkType, "type", "t", "Relates",
		"type of link (for legacy usage)")

	// Register link command with root
	rootCmd.AddCommand(linkCmd)
}

// -----------------------------------------------------------------------------
// Command implementations
// -----------------------------------------------------------------------------

// runLinkLegacy handles the legacy command format: jira-cli link PROJ-1 PROJ-2 --type X
// It delegates to runLinkCreate if two issue keys are provided.
func runLinkLegacy(cmd *cobra.Command, args []string) error {
	// If exactly 2 args that look like issue keys, treat as legacy create
	if len(args) == 2 && looksLikeIssueKey(args[0]) && looksLikeIssueKey(args[1]) {
		if verbose {
			fmt.Println("(Using legacy link syntax - consider using 'jira-cli link create' instead)")
		}
		return runLinkCreate(cmd, args)
	}

	// Otherwise, show help
	return cmd.Help()
}

// runLinkCreate creates a link between two issues
func runLinkCreate(cmd *cobra.Command, args []string) error {
	inwardKey := strings.ToUpper(args[0])
	outwardKey := strings.ToUpper(args[1])

	if verbose {
		fmt.Printf("Linking %s to %s with type '%s'\n", inwardKey, outwardKey, linkType)
	}

	// Create link service
	linkService := jira.NewLinkService(jiraClient)

	// Create the link
	if err := linkService.CreateIssueLink(inwardKey, outwardKey, linkType); err != nil {
		return fmt.Errorf("failed to link issues: %w", err)
	}

	if jsonOutput {
		return outputJSON(map[string]interface{}{
			"status":      "success",
			"message":     fmt.Sprintf("Successfully linked %s to %s", inwardKey, outwardKey),
			"type":        linkType,
			"inwardIssue": inwardKey,
			"outwardIssue": outwardKey,
		})
	}

	fmt.Printf("✓ Successfully linked %s to %s (type: %s)\n", inwardKey, outwardKey, linkType)
	return nil
}

// runLinkTypes lists all available link types
func runLinkTypes(cmd *cobra.Command, args []string) error {
	if verbose {
		fmt.Println("Fetching available link types...")
	}

	// Create link service
	linkService := jira.NewLinkService(jiraClient)

	// Get available link types
	linkTypes, err := linkService.GetAvailableLinkTypes()
	if err != nil {
		return fmt.Errorf("failed to get link types: %w", err)
	}

	if jsonOutput {
		return outputJSON(linkTypes)
	}

	// Display as formatted table
	if len(linkTypes) == 0 {
		fmt.Println("No link types found.")
		return nil
	}

	fmt.Println("Available Link Types:")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("  %-20s %-25s %-25s\n", "Name", "Inward", "Outward")
	fmt.Printf("  %-20s %-25s %-25s\n", strings.Repeat("─", 18), strings.Repeat("─", 23), strings.Repeat("─", 23))

	for _, lt := range linkTypes {
		fmt.Printf("  %-20s %-25s %-25s\n",
			truncateString(lt.Name, 20),
			truncateString(lt.Inward, 25),
			truncateString(lt.Outward, 25),
		)
	}

	return nil
}

// runLinkList lists all links on an issue
func runLinkList(cmd *cobra.Command, args []string) error {
	issueKey := strings.ToUpper(args[0])

	if verbose {
		fmt.Printf("Fetching links for issue %s...\n", issueKey)
	}

	// Create link service
	linkService := jira.NewLinkService(jiraClient)

	// Get issue links
	links, err := linkService.GetIssueLinks(issueKey)
	if err != nil {
		return fmt.Errorf("failed to get links: %w", err)
	}

	if jsonOutput {
		return outputJSON(map[string]interface{}{
			"issueKey": issueKey,
			"links":    links,
			"total":    len(links),
		})
	}

	// Display as formatted table
	if len(links) == 0 {
		fmt.Printf("No links found on issue %s\n", issueKey)
		return nil
	}

	fmt.Printf("Links for %s (%d total):\n", issueKey, len(links))
	fmt.Println(strings.Repeat("─", 95))
	fmt.Printf("  %-8s %-10s %-18s %-12s %-14s %s\n",
		"ID", "Direction", "Type", "Issue", "Status", "Summary")
	fmt.Printf("  %-8s %-10s %-18s %-12s %-14s %s\n",
		strings.Repeat("─", 6), strings.Repeat("─", 8), strings.Repeat("─", 16),
		strings.Repeat("─", 10), strings.Repeat("─", 12), strings.Repeat("─", 20))

	for _, link := range links {
		var direction, relationType, linkedKey, status, summary string

		if link.OutwardIssue != nil {
			// This issue -> outward issue (outward relationship)
			direction = "→"
			relationType = link.Type.Outward
			linkedKey = link.OutwardIssue.Key
			status, summary = extractIssueDetails(link.OutwardIssue.Fields)
		} else if link.InwardIssue != nil {
			// Inward issue -> this issue (inward relationship)
			direction = "←"
			relationType = link.Type.Inward
			linkedKey = link.InwardIssue.Key
			status, summary = extractIssueDetails(link.InwardIssue.Fields)
		}

		fmt.Printf("  %-8s %-10s %-18s %-12s %-14s %s\n",
			truncateString(link.ID, 8),
			direction,
			truncateString(relationType, 18),
			truncateString(linkedKey, 12),
			truncateString(fmt.Sprintf("[%s]", status), 14),
			truncateString(summary, 30),
		)
	}

	return nil
}

// runLinkDelete deletes a link by ID
func runLinkDelete(cmd *cobra.Command, args []string) error {
	linkID := args[0]

	if !linkConfirm {
		return fmt.Errorf("deletion requires --confirm flag for safety")
	}

	if verbose {
		fmt.Printf("Deleting link %s...\n", linkID)
	}

	// Create link service
	linkService := jira.NewLinkService(jiraClient)

	// Delete the link
	if err := linkService.DeleteIssueLink(linkID); err != nil {
		return fmt.Errorf("failed to delete link: %w", err)
	}

	if jsonOutput {
		return outputJSON(map[string]interface{}{
			"status":  "success",
			"message": fmt.Sprintf("Successfully deleted link %s", linkID),
			"linkId":  linkID,
		})
	}

	fmt.Printf("✓ Successfully deleted link %s\n", linkID)
	return nil
}

// -----------------------------------------------------------------------------
// Helper functions
// -----------------------------------------------------------------------------

// truncateString truncates a string to maxLen runes, adding "..." if truncated.
// Uses rune count instead of byte length to properly handle Unicode characters.
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

// extractIssueDetails extracts status and summary from issue fields map
func extractIssueDetails(fields map[string]interface{}) (status, summary string) {
	if fields == nil {
		return "", ""
	}

	// Extract status
	if statusMap, ok := fields["status"].(map[string]interface{}); ok {
		if name, ok := statusMap["name"].(string); ok {
			status = name
		}
	}

	// Extract summary
	if s, ok := fields["summary"].(string); ok {
		summary = s
	}

	return status, summary
}
