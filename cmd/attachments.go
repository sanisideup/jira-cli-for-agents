package cmd

import (
	"fmt"
	"strings"

	"github.com/sanisideup/jira-cli-for-agents/pkg/jira"
	"github.com/sanisideup/jira-cli-for-agents/pkg/models"
	"github.com/spf13/cobra"
)

var (
	attachmentOutput     string
	attachmentConfirm    bool
	attachmentForce      bool
	attachmentNoProgress bool
)

// attachmentsCmd is the parent command for attachment operations
var attachmentsCmd = &cobra.Command{
	Use:     "attachment <subcommand>",
	Aliases: []string{"attachments"},
	Short:   "Manage file attachments on Jira issues",
	Long: `Manage file attachments on Jira issues with various operations.

Subcommands:
  list     - List all attachments on an issue
  upload   - Upload one or more files to an issue
  download - Download an attachment
  delete   - Delete an attachment

Examples:
  jcfa attachment list PROJ-123
  jcfa attachment upload PROJ-123 file.pdf
  jcfa attachment download PROJ-123 file.pdf
  jcfa attachment delete 10001 --confirm`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// attachmentListCmd lists all attachments on an issue
var attachmentListCmd = &cobra.Command{
	Use:   "list <issue-key>",
	Short: "List all attachments on an issue",
	Long: `List all file attachments on an issue with metadata.

Examples:
  jcfa attachment list PROJ-123
  jcfa attachment list PROJ-123 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAttachmentList,
}

// attachmentUploadCmd uploads files to an issue
var attachmentUploadCmd = &cobra.Command{
	Use:   "upload <issue-key> <file-path> [file-path...]",
	Short: "Upload one or more files to an issue",
	Long: `Upload one or more file attachments to an issue.

Files larger than 1MB will show a progress bar during upload.
Use --no-progress to disable the progress bar.

Examples:
  jcfa attachment upload PROJ-123 design.pdf
  jcfa attachment upload PROJ-123 screenshot1.png screenshot2.png
  jcfa attachment upload PROJ-123 large-file.zip --no-progress`,
	Args: cobra.MinimumNArgs(2),
	RunE: runAttachmentUpload,
}

// attachmentDownloadCmd downloads an attachment
var attachmentDownloadCmd = &cobra.Command{
	Use:   "download <issue-key> <attachment-id-or-filename>",
	Short: "Download an attachment",
	Long: `Download an attachment by ID or filename.

If a filename is provided, the command will search for a matching attachment.
Use --output to specify the destination path (file or directory).

Examples:
  jcfa attachment download PROJ-123 10001
  jcfa attachment download PROJ-123 design.pdf
  jcfa attachment download PROJ-123 design.pdf --output ./downloads/
  jcfa attachment download PROJ-123 10001 --output custom-name.pdf`,
	Args: cobra.ExactArgs(2),
	RunE: runAttachmentDownload,
}

// attachmentDeleteCmd deletes an attachment
var attachmentDeleteCmd = &cobra.Command{
	Use:   "delete <attachment-id>",
	Short: "Delete an attachment",
	Long: `Delete an attachment by ID.

Requires --confirm flag for safety.
Note: You need appropriate permissions to delete attachments.

Examples:
  jcfa attachment delete 10001 --confirm
  jcfa attachment delete 10002 --confirm --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAttachmentDelete,
}

func init() {
	// Add subcommands
	attachmentsCmd.AddCommand(attachmentListCmd)
	attachmentsCmd.AddCommand(attachmentUploadCmd)
	attachmentsCmd.AddCommand(attachmentDownloadCmd)
	attachmentsCmd.AddCommand(attachmentDeleteCmd)

	// Add flags
	attachmentDownloadCmd.Flags().StringVar(&attachmentOutput, "output", "", "Output path (file or directory)")
	attachmentUploadCmd.Flags().BoolVar(&attachmentNoProgress, "no-progress", false, "Disable progress bar")
	attachmentDownloadCmd.Flags().BoolVar(&attachmentNoProgress, "no-progress", false, "Disable progress bar")

	attachmentDeleteCmd.Flags().BoolVar(&attachmentConfirm, "confirm", false, "Confirm deletion")
	attachmentDeleteCmd.MarkFlagRequired("confirm")

	// Register command
	rootCmd.AddCommand(attachmentsCmd)
}

func runAttachmentList(cmd *cobra.Command, args []string) error {
	issueKey := args[0]

	if verbose {
		fmt.Printf("Listing attachments for issue %s\n", issueKey)
	}

	// Create attachment service
	attachmentService := jira.NewAttachmentService(jiraClient)

	// List attachments
	attachments, err := attachmentService.ListAttachments(issueKey)
	if err != nil {
		return fmt.Errorf("failed to list attachments: %w", err)
	}

	if jsonOutput {
		return outputJSON(attachments)
	}

	// Display attachments
	if len(attachments) == 0 {
		fmt.Printf("No attachments found on issue %s\n", issueKey)
		return nil
	}

	fmt.Printf("Attachments for %s (%d total):\n\n", issueKey, len(attachments))
	fmt.Printf("%-10s %-30s %-10s %-20s %s\n", "ID", "Filename", "Size", "Author", "Date")
	fmt.Println(strings.Repeat("-", 90))

	for _, att := range attachments {
		filename := att.Filename
		if len(filename) > 30 {
			filename = filename[:27] + "..."
		}

		author := att.Author.DisplayName
		if len(author) > 20 {
			author = author[:17] + "..."
		}

		fmt.Printf("%-10s %-30s %-10s %-20s %s\n",
			att.ID,
			filename,
			jira.FormatFileSize(att.Size),
			author,
			jira.FormatDate(att.Created),
		)
	}

	return nil
}

func runAttachmentUpload(cmd *cobra.Command, args []string) error {
	issueKey := args[0]
	filePaths := args[1:]

	if verbose {
		fmt.Printf("Uploading %d file(s) to issue %s\n", len(filePaths), issueKey)
	}

	// Create attachment service
	attachmentService := jira.NewAttachmentService(jiraClient)

	// Upload each file
	uploaded := []string{}
	failed := []string{}

	showProgress := !attachmentNoProgress

	for _, filePath := range filePaths {
		if verbose {
			fmt.Printf("Uploading %s...\n", filePath)
		}

		attachment, err := attachmentService.UploadAttachment(issueKey, filePath, showProgress)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", filePath, err))
			continue
		}

		uploaded = append(uploaded, attachment.Filename)
	}

	// Display results
	if jsonOutput {
		return outputJSON(map[string]interface{}{
			"uploaded": len(uploaded),
			"failed":   len(failed),
			"files":    uploaded,
			"errors":   failed,
		})
	}

	if len(uploaded) > 0 {
		fmt.Printf("\n✓ Successfully uploaded %d file(s):\n", len(uploaded))
		for _, filename := range uploaded {
			fmt.Printf("  - %s\n", filename)
		}
	}

	if len(failed) > 0 {
		fmt.Printf("\n✗ Failed to upload %d file(s):\n", len(failed))
		for _, errMsg := range failed {
			fmt.Printf("  - %s\n", errMsg)
		}
		return fmt.Errorf("some uploads failed")
	}

	return nil
}

func runAttachmentDownload(cmd *cobra.Command, args []string) error {
	issueKey := args[0]
	attachmentIDOrFilename := args[1]

	if verbose {
		fmt.Printf("Downloading attachment '%s' from issue %s\n", attachmentIDOrFilename, issueKey)
	}

	// Create attachment service
	attachmentService := jira.NewAttachmentService(jiraClient)

	// First, try to find the attachment
	var attachment *models.Attachment

	// Try as filename first
	att, err := attachmentService.FindAttachmentByFilename(issueKey, attachmentIDOrFilename)
	if err == nil {
		attachment = att
	} else {
		// Try getting by ID from the list
		attachments, listErr := attachmentService.ListAttachments(issueKey)
		if listErr != nil {
			return fmt.Errorf("failed to find attachment: %w", listErr)
		}

		// Search for matching ID
		for _, a := range attachments {
			if a.ID == attachmentIDOrFilename {
				attachment = &a
				break
			}
		}

		if attachment == nil {
			return fmt.Errorf("attachment '%s' not found. Run 'jcfa attachment list %s' to see available attachments", attachmentIDOrFilename, issueKey)
		}
	}

	// Download the attachment
	showProgress := !attachmentNoProgress
	err = attachmentService.DownloadAttachment(attachment, attachmentOutput, showProgress)
	if err != nil {
		return fmt.Errorf("failed to download attachment: %w", err)
	}

	if jsonOutput {
		return outputJSON(map[string]string{
			"status":   "success",
			"filename": attachment.Filename,
			"size":     jira.FormatFileSize(attachment.Size),
		})
	}

	outputPath := attachmentOutput
	if outputPath == "" {
		outputPath = attachment.Filename
	}

	fmt.Printf("\n✓ Successfully downloaded %s (%s) to %s\n",
		attachment.Filename,
		jira.FormatFileSize(attachment.Size),
		outputPath,
	)

	return nil
}

func runAttachmentDelete(cmd *cobra.Command, args []string) error {
	attachmentID := args[0]

	if !attachmentConfirm {
		return fmt.Errorf("deletion requires --confirm flag for safety")
	}

	if verbose {
		fmt.Printf("Deleting attachment %s\n", attachmentID)
	}

	// Create attachment service
	attachmentService := jira.NewAttachmentService(jiraClient)

	// Delete attachment
	err := attachmentService.DeleteAttachment(attachmentID)
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	if jsonOutput {
		return outputJSON(map[string]string{
			"status":  "success",
			"message": "Attachment deleted successfully",
		})
	}

	fmt.Printf("✓ Successfully deleted attachment %s\n", attachmentID)

	return nil
}
