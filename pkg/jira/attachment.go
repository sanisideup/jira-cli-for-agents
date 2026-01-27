package jira

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sanisideup/jira-cli-for-agents/pkg/client"
	"github.com/sanisideup/jira-cli-for-agents/pkg/models"
	"github.com/schollz/progressbar/v3"
)

// AttachmentService handles file attachment operations
type AttachmentService struct {
	client *client.Client
}

// NewAttachmentService creates a new attachment service
func NewAttachmentService(client *client.Client) *AttachmentService {
	return &AttachmentService{client: client}
}

// ListAttachments retrieves all attachments for an issue
// Parameters:
//   - issueKey: The issue key (e.g., "PROJ-123")
// Returns list of attachments with metadata
func (s *AttachmentService) ListAttachments(issueKey string) ([]models.Attachment, error) {
	if issueKey == "" {
		return nil, fmt.Errorf("issue key cannot be empty")
	}

	var issue models.Issue
	var errorResp models.ErrorResponse

	// Fetch issue with only attachment field
	resp, err := s.client.HTTPClient.R().
		SetQueryParam("fields", "attachment").
		SetResult(&issue).
		SetError(&errorResp).
		Get(fmt.Sprintf("/issue/%s", issueKey))

	if err != nil {
		return nil, fmt.Errorf("failed to list attachments for %s: %w", issueKey, err)
	}

	if resp.IsError() {
		if resp.StatusCode() == 404 {
			return nil, fmt.Errorf("issue '%s' not found", issueKey)
		}
		return nil, fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	// Extract attachments from fields
	attachmentsField, ok := issue.Fields["attachment"]
	if !ok {
		return []models.Attachment{}, nil
	}

	// Convert to attachment slice
	var attachments []models.Attachment
	if attachmentList, ok := attachmentsField.([]interface{}); ok {
		for _, item := range attachmentList {
			if attMap, ok := item.(map[string]interface{}); ok {
				attachment := parseAttachment(attMap)
				attachments = append(attachments, attachment)
			}
		}
	}

	return attachments, nil
}

// UploadAttachment uploads a file to an issue
// Parameters:
//   - issueKey: The issue key (e.g., "PROJ-123")
//   - filePath: Path to the file to upload
//   - showProgress: Whether to show progress bar for large files
// Returns the created attachment
func (s *AttachmentService) UploadAttachment(issueKey, filePath string, showProgress bool) (*models.Attachment, error) {
	if issueKey == "" {
		return nil, fmt.Errorf("issue key cannot be empty")
	}

	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// Validate file exists and is readable
	if err := ValidateFilePath(filePath); err != nil {
		return nil, err
	}

	// Get file info for size check
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Prepare reader (with or without progress bar)
	var reader io.Reader = file
	if showProgress && fileInfo.Size() > 1024*1024 { // Show progress for files > 1MB
		bar := progressbar.DefaultBytes(
			fileInfo.Size(),
			fmt.Sprintf("Uploading %s", filepath.Base(filePath)),
		)
		// Use pointer to progressbar.Reader to implement io.Reader
		pbReader := progressbar.NewReader(file, bar)
		reader = &pbReader
	}

	var result []models.Attachment
	var errorResp models.ErrorResponse

	resp, err := s.client.HTTPClient.R().
		SetHeader("X-Atlassian-Token", "no-check").
		SetFileReader("file", filepath.Base(filePath), reader).
		SetResult(&result).
		SetError(&errorResp).
		Post(fmt.Sprintf("/issue/%s/attachments", issueKey))

	if err != nil {
		return nil, fmt.Errorf("failed to upload attachment: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no attachment returned from API")
	}

	return &result[0], nil
}

// DownloadAttachment downloads an attachment to a local file
// Parameters:
//   - attachment: The attachment metadata (must have Content URL)
//   - outputPath: Path where file should be saved (file or directory)
//   - showProgress: Whether to show progress bar for large files
// Returns error if download fails
func (s *AttachmentService) DownloadAttachment(attachment *models.Attachment, outputPath string, showProgress bool) error {
	if attachment == nil {
		return fmt.Errorf("attachment cannot be nil")
	}

	if attachment.Content == "" {
		return fmt.Errorf("attachment has no download URL")
	}

	// Determine output file path
	var targetPath string
	if outputPath == "" {
		targetPath = attachment.Filename
	} else {
		// Check if outputPath is a directory
		if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
			targetPath = filepath.Join(outputPath, attachment.Filename)
		} else {
			targetPath = outputPath
		}
	}

	// Create output file
	outFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Download the file
	resp, err := s.client.HTTPClient.R().
		SetDoNotParseResponse(true).
		Get(attachment.Content)

	if err != nil {
		return fmt.Errorf("failed to download attachment: %w", err)
	}
	defer resp.RawBody().Close()

	if resp.IsError() {
		return fmt.Errorf("download failed with status: %s", resp.Status())
	}

	// Copy with or without progress bar
	if showProgress && attachment.Size > 1024*1024 { // Show progress for files > 1MB
		bar := progressbar.DefaultBytes(
			attachment.Size,
			fmt.Sprintf("Downloading %s", attachment.Filename),
		)
		_, err = io.Copy(io.MultiWriter(outFile, bar), resp.RawBody())
	} else {
		_, err = io.Copy(outFile, resp.RawBody())
	}

	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// DeleteAttachment deletes an attachment by ID
// Parameters:
//   - attachmentID: The attachment ID to delete
// Returns error if deletion fails (e.g., insufficient permissions)
func (s *AttachmentService) DeleteAttachment(attachmentID string) error {
	if attachmentID == "" {
		return fmt.Errorf("attachment ID cannot be empty")
	}

	var errorResp models.ErrorResponse

	resp, err := s.client.HTTPClient.R().
		SetError(&errorResp).
		Delete(fmt.Sprintf("/attachment/%s", attachmentID))

	if err != nil {
		return fmt.Errorf("failed to delete attachment %s: %w", attachmentID, err)
	}

	if resp.IsError() {
		if resp.StatusCode() == 404 {
			return fmt.Errorf("attachment '%s' not found", attachmentID)
		}
		if resp.StatusCode() == 403 {
			return fmt.Errorf("you don't have permission to delete this attachment")
		}
		return fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return nil
}

// FindAttachmentByFilename searches for an attachment by filename
// Parameters:
//   - issueKey: The issue key (e.g., "PROJ-123")
//   - filename: The filename to search for
// Returns the matching attachment or error if not found
func (s *AttachmentService) FindAttachmentByFilename(issueKey, filename string) (*models.Attachment, error) {
	attachments, err := s.ListAttachments(issueKey)
	if err != nil {
		return nil, err
	}

	for _, att := range attachments {
		if att.Filename == filename {
			return &att, nil
		}
	}

	return nil, fmt.Errorf("attachment '%s' not found on issue '%s'", filename, issueKey)
}

// parseAttachment converts a map to Attachment struct
func parseAttachment(data map[string]interface{}) models.Attachment {
	att := models.Attachment{}

	if self, ok := data["self"].(string); ok {
		att.Self = self
	}
	if id, ok := data["id"].(string); ok {
		att.ID = id
	}
	if filename, ok := data["filename"].(string); ok {
		att.Filename = filename
	}
	if created, ok := data["created"].(string); ok {
		att.Created = created
	}
	if size, ok := data["size"].(float64); ok {
		att.Size = int64(size)
	}
	if mimeType, ok := data["mimeType"].(string); ok {
		att.MimeType = mimeType
	}
	if content, ok := data["content"].(string); ok {
		att.Content = content
	}
	if thumbnail, ok := data["thumbnail"].(string); ok {
		att.Thumbnail = thumbnail
	}

	// Parse author if present
	if author, ok := data["author"].(map[string]interface{}); ok {
		if displayName, ok := author["displayName"].(string); ok {
			att.Author.DisplayName = displayName
		}
		if accountId, ok := author["accountId"].(string); ok {
			att.Author.AccountID = accountId
		}
	}

	return att
}
