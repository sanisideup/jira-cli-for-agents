package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sanisideup/jira-cli/pkg/jira"
	"github.com/spf13/cobra"
)

// Command flags
var (
	showLinks    bool
	showSubtasks bool
	showComments bool
	showFull     bool
)

var getCmd = &cobra.Command{
	Use:   "get <issue-key>",
	Short: "Get details of a Jira issue",
	Long: `Retrieves and displays detailed information about a specific Jira issue.

By default, shows core fields plus description and attachments.
Use flags to display additional sections like linked issues, subtasks, and comments.

Examples:
  # Default output (description + attachments)
  jira-cli get PROJ-123

  # With linked issues
  jira-cli get PROJ-123 --links
  jira-cli get PROJ-123 -l

  # With subtasks
  jira-cli get PROJ-123 --subtasks
  jira-cli get PROJ-123 -s

  # With comments
  jira-cli get PROJ-123 --comments
  jira-cli get PROJ-123 -c

  # Combine multiple options
  jira-cli get PROJ-123 --links --comments
  jira-cli get PROJ-123 -lc

  # Show everything
  jira-cli get PROJ-123 --full
  jira-cli get PROJ-123 -f

  # JSON output
  jira-cli get PROJ-123 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runGet,
}

func init() {
	rootCmd.AddCommand(getCmd)

	// Add command-specific flags
	getCmd.Flags().BoolVarP(&showLinks, "links", "l", false, "Show linked issues")
	getCmd.Flags().BoolVarP(&showSubtasks, "subtasks", "s", false, "Show subtasks")
	getCmd.Flags().BoolVarP(&showComments, "comments", "c", false, "Show comments")
	getCmd.Flags().BoolVarP(&showFull, "full", "f", false, "Show all details (links + subtasks + comments)")
}

func runGet(cmd *cobra.Command, args []string) error {
	issueKey := args[0]

	// If --full flag is set, enable all optional sections
	if showFull {
		showLinks = true
		showSubtasks = true
		showComments = true
	}

	// Create search service
	searchService := jira.NewSearchService(jiraClient)

	// Get the issue
	issue, err := searchService.GetIssue(issueKey)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	// Output based on format
	if jsonOutput {
		return outputJSON(issue)
	}

	return outputHumanReadable(issue, issueKey)
}

// outputJSON outputs the issue in JSON format
func outputJSON(issue interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(issue); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

// outputHumanReadable outputs the issue in a human-readable format
func outputHumanReadable(issue interface{}, issueKey string) error {
	// Type assert to access fields
	issueData, ok := issue.(map[string]interface{})
	if !ok {
		// If it's not a map, try to convert via JSON
		jsonBytes, err := json.Marshal(issue)
		if err != nil {
			return fmt.Errorf("failed to marshal issue: %w", err)
		}
		if err := json.Unmarshal(jsonBytes, &issueData); err != nil {
			return fmt.Errorf("failed to unmarshal issue: %w", err)
		}
	}

	// Extract issue key
	key, _ := issueData["key"].(string)

	// Extract fields
	fields, ok := issueData["fields"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid issue format: missing fields")
	}

	// Extract common fields
	summary, _ := fields["summary"].(string)

	// Issue type
	var issueType string
	if it, ok := fields["issuetype"].(map[string]interface{}); ok {
		issueType, _ = it["name"].(string)
	}

	// Status
	var status string
	if st, ok := fields["status"].(map[string]interface{}); ok {
		status, _ = st["name"].(string)
	}

	// Priority
	var priority string = "Unassigned"
	if pr, ok := fields["priority"].(map[string]interface{}); ok {
		priority, _ = pr["name"].(string)
	}

	// Assignee
	var assignee string = "Unassigned"
	if as, ok := fields["assignee"].(map[string]interface{}); ok {
		assignee, _ = as["displayName"].(string)
	}

	// Created and updated dates
	var created, updated string
	if cr, ok := fields["created"].(string); ok {
		if t, err := time.Parse(time.RFC3339, cr); err == nil {
			created = t.Format("2006-01-02")
		} else {
			created = cr
		}
	}
	if up, ok := fields["updated"].(string); ok {
		if t, err := time.Parse(time.RFC3339, up); err == nil {
			updated = t.Format("2006-01-02")
		} else {
			updated = up
		}
	}

	// Print header with separator
	fmt.Printf("%s: %s\n", key, summary)
	fmt.Println(strings.Repeat("=", 80))

	// Print core fields in two columns
	fmt.Printf("%-40s %s\n", fmt.Sprintf("Type: %s", issueType), fmt.Sprintf("Status: %s", status))
	fmt.Printf("%-40s %s\n", fmt.Sprintf("Priority: %s", priority), fmt.Sprintf("Assignee: %s", assignee))
	fmt.Printf("%-40s %s\n", fmt.Sprintf("Created: %s", created), fmt.Sprintf("Updated: %s", updated))

	// Print Epic Link if present
	printEpicLink(fields)

	// Print Labels if present
	printLabels(fields)

	// Print Description (using ADF parser)
	printDescription(fields)

	// Print Attachments (always shown by default)
	printAttachments(fields)

	// Print Linked Issues (if --links or --full flag)
	if showLinks {
		printLinkedIssues(fields)
	}

	// Print Subtasks (if --subtasks or --full flag)
	if showSubtasks {
		printSubtasks(fields)
	}

	// Print Comments (if --comments or --full flag)
	if showComments {
		if err := printComments(issueKey); err != nil {
			// Non-fatal: just print warning and continue
			fmt.Printf("\n[Warning: Could not fetch comments: %v]\n", err)
		}
	}

	return nil
}

// printEpicLink prints the epic link custom field if present
func printEpicLink(fields map[string]interface{}) {
	// Try common epic link field IDs
	epicFieldIDs := []string{"customfield_10014", "customfield_10008", "parent"}

	for _, fieldID := range epicFieldIDs {
		if value, exists := fields[fieldID]; exists && value != nil {
			switch v := value.(type) {
			case string:
				if v != "" {
					fmt.Printf("Epic Link: %s\n", v)
					return
				}
			case map[string]interface{}:
				if key, ok := v["key"].(string); ok {
					fmt.Printf("Epic Link: %s\n", key)
					return
				}
			}
		}
	}
}

// printLabels prints labels if present
func printLabels(fields map[string]interface{}) {
	if lb, ok := fields["labels"].([]interface{}); ok && len(lb) > 0 {
		labels := make([]string, 0, len(lb))
		for _, l := range lb {
			if labelStr, ok := l.(string); ok {
				labels = append(labels, labelStr)
			}
		}
		if len(labels) > 0 {
			fmt.Printf("Labels: %s\n", strings.Join(labels, ", "))
		}
	}
}

// printDescription prints the description, parsing ADF format
func printDescription(fields map[string]interface{}) {
	description := fields["description"]

	if description == nil {
		return
	}

	// Use ADF parser to convert to plain text
	plainText := jira.ADFToPlainText(description)

	if plainText == "" {
		return
	}

	fmt.Println()
	fmt.Println("Description:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println(plainText)
}

// printAttachments prints the list of attachments
func printAttachments(fields map[string]interface{}) {
	attachments, ok := fields["attachment"].([]interface{})
	if !ok || len(attachments) == 0 {
		return
	}

	fmt.Println()
	fmt.Printf("Attachments (%d):\n", len(attachments))
	fmt.Println(strings.Repeat("-", 80))

	for _, att := range attachments {
		attMap, ok := att.(map[string]interface{})
		if !ok {
			continue
		}

		filename, _ := attMap["filename"].(string)
		size, _ := attMap["size"].(float64)
		created, _ := attMap["created"].(string)

		// Get author display name
		var author string = "Unknown"
		if authorMap, ok := attMap["author"].(map[string]interface{}); ok {
			author, _ = authorMap["displayName"].(string)
		}

		// Format size
		sizeStr := jira.FormatFileSize(int64(size))

		// Format date
		dateStr := jira.FormatDate(created)
		if len(dateStr) > 10 {
			dateStr = dateStr[:10] // Just the date part
		}

		fmt.Printf("  %-35s %10s    %-15s %s\n", filename, sizeStr, author, dateStr)
	}
}

// printLinkedIssues prints linked issues
func printLinkedIssues(fields map[string]interface{}) {
	issuelinks, ok := fields["issuelinks"].([]interface{})
	if !ok || len(issuelinks) == 0 {
		fmt.Println()
		fmt.Println("Linked Issues (0):")
		fmt.Println(strings.Repeat("-", 80))
		fmt.Println("  No linked issues")
		return
	}

	fmt.Println()
	fmt.Printf("Linked Issues (%d):\n", len(issuelinks))
	fmt.Println(strings.Repeat("-", 80))

	for _, link := range issuelinks {
		linkMap, ok := link.(map[string]interface{})
		if !ok {
			continue
		}

		// Get link type
		linkType, _ := linkMap["type"].(map[string]interface{})
		inward, _ := linkType["inward"].(string)
		outward, _ := linkType["outward"].(string)

		// Determine direction and linked issue
		var direction, linkName, linkedKey, linkedStatus, linkedSummary string

		if outwardIssue, ok := linkMap["outwardIssue"].(map[string]interface{}); ok {
			direction = "\u2192" // → arrow
			linkName = outward
			linkedKey, _ = outwardIssue["key"].(string)

			if issueFields, ok := outwardIssue["fields"].(map[string]interface{}); ok {
				if statusMap, ok := issueFields["status"].(map[string]interface{}); ok {
					linkedStatus, _ = statusMap["name"].(string)
				}
				linkedSummary, _ = issueFields["summary"].(string)
			}
		} else if inwardIssue, ok := linkMap["inwardIssue"].(map[string]interface{}); ok {
			direction = "\u2190" // <- arrow
			linkName = inward
			linkedKey, _ = inwardIssue["key"].(string)

			if issueFields, ok := inwardIssue["fields"].(map[string]interface{}); ok {
				if statusMap, ok := issueFields["status"].(map[string]interface{}); ok {
					linkedStatus, _ = statusMap["name"].(string)
				}
				linkedSummary, _ = issueFields["summary"].(string)
			}
		}

		// Truncate summary if too long
		if len(linkedSummary) > 40 {
			linkedSummary = linkedSummary[:37] + "..."
		}

		fmt.Printf("  %s %-12s %-12s [%-12s] %s\n", direction, linkName, linkedKey, linkedStatus, linkedSummary)
	}
}

// printSubtasks prints subtasks
func printSubtasks(fields map[string]interface{}) {
	subtasks, ok := fields["subtasks"].([]interface{})
	if !ok || len(subtasks) == 0 {
		fmt.Println()
		fmt.Println("Subtasks (0):")
		fmt.Println(strings.Repeat("-", 80))
		fmt.Println("  No subtasks")
		return
	}

	fmt.Println()
	fmt.Printf("Subtasks (%d):\n", len(subtasks))
	fmt.Println(strings.Repeat("-", 80))

	for _, subtask := range subtasks {
		subtaskMap, ok := subtask.(map[string]interface{})
		if !ok {
			continue
		}

		subtaskKey, _ := subtaskMap["key"].(string)

		var subtaskStatus, subtaskSummary string
		if subtaskFields, ok := subtaskMap["fields"].(map[string]interface{}); ok {
			if statusMap, ok := subtaskFields["status"].(map[string]interface{}); ok {
				subtaskStatus, _ = statusMap["name"].(string)
			}
			subtaskSummary, _ = subtaskFields["summary"].(string)
		}

		// Truncate summary if too long
		if len(subtaskSummary) > 50 {
			subtaskSummary = subtaskSummary[:47] + "..."
		}

		fmt.Printf("  %-15s [%-12s] %s\n", subtaskKey, subtaskStatus, subtaskSummary)
	}
}

// printComments fetches and prints comments (requires additional API call)
func printComments(issueKey string) error {
	commentService := jira.NewCommentService(jiraClient)

	// Fetch comments in ascending order (oldest first)
	commentsResp, err := commentService.ListComments(issueKey, "created")
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Comments (%d):\n", commentsResp.Total)
	fmt.Println(strings.Repeat("-", 80))

	if len(commentsResp.Comments) == 0 {
		fmt.Println("  No comments")
		return nil
	}

	for i, comment := range commentsResp.Comments {
		// Format date
		dateStr := jira.FormatDate(comment.Created)

		// Get author name
		authorName := comment.Author.DisplayName
		if authorName == "" {
			authorName = "Unknown"
		}

		// Parse comment body (ADF format)
		bodyText := jira.ADFToPlainText(comment.Body)

		fmt.Printf("[%s] %s:\n", dateStr, authorName)
		fmt.Println(bodyText)

		// Add separator between comments (except last)
		if i < len(commentsResp.Comments)-1 {
			fmt.Println()
		}
	}

	return nil
}

// printCustomField prints a custom field if it exists and is not empty
func printCustomField(fields map[string]interface{}, fieldID, fieldName string) {
	if value, exists := fields[fieldID]; exists && value != nil {
		// Handle different value types
		switch v := value.(type) {
		case string:
			if v != "" {
				fmt.Printf("%s: %s\n", fieldName, v)
			}
		case float64:
			fmt.Printf("%s: %.0f\n", fieldName, v)
		case int:
			fmt.Printf("%s: %d\n", fieldName, v)
		case map[string]interface{}:
			// For objects like epic link, try to get the key
			if key, ok := v["key"].(string); ok {
				fmt.Printf("%s: %s\n", fieldName, key)
			} else if name, ok := v["name"].(string); ok {
				fmt.Printf("%s: %s\n", fieldName, name)
			}
		}
	}
}
