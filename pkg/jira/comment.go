package jira

import (
	"fmt"

	"github.com/sanisideup/jira-cli-for-agents/pkg/client"
	"github.com/sanisideup/jira-cli-for-agents/pkg/models"
)

// CommentService handles comment operations on Jira issues
type CommentService struct {
	client *client.Client
}

// NewCommentService creates a new comment service
func NewCommentService(client *client.Client) *CommentService {
	return &CommentService{client: client}
}

// AddComment adds a comment to an issue
// Parameters:
//   - issueKey: The issue key (e.g., "PROJ-123")
//   - text: The comment text (will be converted to ADF format)
// Returns the created comment
func (s *CommentService) AddComment(issueKey, text string) (*models.Comment, error) {
	if issueKey == "" {
		return nil, fmt.Errorf("issue key cannot be empty")
	}

	if text == "" {
		return nil, fmt.Errorf("comment text cannot be empty")
	}

	// Convert plain text to ADF (Atlassian Document Format)
	body := map[string]interface{}{
		"body": map[string]interface{}{
			"type":    "doc",
			"version": 1,
			"content": []map[string]interface{}{
				{
					"type": "paragraph",
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": text,
						},
					},
				},
			},
		},
	}

	var comment models.Comment
	var errorResp models.ErrorResponse

	resp, err := s.client.HTTPClient.R().
		SetBody(body).
		SetResult(&comment).
		SetError(&errorResp).
		Post(fmt.Sprintf("/issue/%s/comment", issueKey))

	if err != nil {
		return nil, fmt.Errorf("failed to add comment to %s: %w", issueKey, err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return &comment, nil
}

// ListComments retrieves all comments for an issue
// Parameters:
//   - issueKey: The issue key (e.g., "PROJ-123")
//   - orderBy: Sort order ("created" or "-created" for descending)
// Returns paginated list of comments
func (s *CommentService) ListComments(issueKey string, orderBy string) (*models.CommentsResponse, error) {
	if issueKey == "" {
		return nil, fmt.Errorf("issue key cannot be empty")
	}

	// Default to ascending by created date
	if orderBy == "" {
		orderBy = "created"
	}

	var result models.CommentsResponse
	var errorResp models.ErrorResponse

	req := s.client.HTTPClient.R().
		SetResult(&result).
		SetError(&errorResp)

	// Add orderBy query parameter
	req.SetQueryParam("orderBy", orderBy)

	resp, err := req.Get(fmt.Sprintf("/issue/%s/comment", issueKey))

	if err != nil {
		return nil, fmt.Errorf("failed to list comments for %s: %w", issueKey, err)
	}

	if resp.IsError() {
		if resp.StatusCode() == 404 {
			return nil, fmt.Errorf("issue '%s' not found", issueKey)
		}
		return nil, fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return &result, nil
}

// GetComment retrieves a specific comment by ID
// Parameters:
//   - issueKey: The issue key (e.g., "PROJ-123")
//   - commentID: The comment ID
// Returns the comment details
func (s *CommentService) GetComment(issueKey, commentID string) (*models.Comment, error) {
	if issueKey == "" {
		return nil, fmt.Errorf("issue key cannot be empty")
	}

	if commentID == "" {
		return nil, fmt.Errorf("comment ID cannot be empty")
	}

	var comment models.Comment
	var errorResp models.ErrorResponse

	resp, err := s.client.HTTPClient.R().
		SetResult(&comment).
		SetError(&errorResp).
		Get(fmt.Sprintf("/issue/%s/comment/%s", issueKey, commentID))

	if err != nil {
		return nil, fmt.Errorf("failed to get comment %s: %w", commentID, err)
	}

	if resp.IsError() {
		if resp.StatusCode() == 404 {
			return nil, fmt.Errorf("comment '%s' not found on issue '%s'", commentID, issueKey)
		}
		return nil, fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return &comment, nil
}

// UpdateComment updates an existing comment
// Parameters:
//   - issueKey: The issue key (e.g., "PROJ-123")
//   - commentID: The comment ID to update
//   - text: The new comment text (will be converted to ADF format)
// Returns error if update fails (e.g., insufficient permissions)
func (s *CommentService) UpdateComment(issueKey, commentID, text string) error {
	if issueKey == "" {
		return fmt.Errorf("issue key cannot be empty")
	}

	if commentID == "" {
		return fmt.Errorf("comment ID cannot be empty")
	}

	if text == "" {
		return fmt.Errorf("comment text cannot be empty")
	}

	// Convert plain text to ADF
	body := map[string]interface{}{
		"body": map[string]interface{}{
			"type":    "doc",
			"version": 1,
			"content": []map[string]interface{}{
				{
					"type": "paragraph",
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": text,
						},
					},
				},
			},
		},
	}

	var errorResp models.ErrorResponse

	resp, err := s.client.HTTPClient.R().
		SetBody(body).
		SetError(&errorResp).
		Put(fmt.Sprintf("/issue/%s/comment/%s", issueKey, commentID))

	if err != nil {
		return fmt.Errorf("failed to update comment %s: %w", commentID, err)
	}

	if resp.IsError() {
		if resp.StatusCode() == 404 {
			return fmt.Errorf("comment '%s' not found on issue '%s'", commentID, issueKey)
		}
		if resp.StatusCode() == 403 {
			return fmt.Errorf("you don't have permission to update this comment")
		}
		return fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return nil
}

// DeleteComment deletes a comment
// Parameters:
//   - issueKey: The issue key (e.g., "PROJ-123")
//   - commentID: The comment ID to delete
// Returns error if deletion fails (e.g., insufficient permissions)
func (s *CommentService) DeleteComment(issueKey, commentID string) error {
	if issueKey == "" {
		return fmt.Errorf("issue key cannot be empty")
	}

	if commentID == "" {
		return fmt.Errorf("comment ID cannot be empty")
	}

	var errorResp models.ErrorResponse

	resp, err := s.client.HTTPClient.R().
		SetError(&errorResp).
		Delete(fmt.Sprintf("/issue/%s/comment/%s", issueKey, commentID))

	if err != nil {
		return fmt.Errorf("failed to delete comment %s: %w", commentID, err)
	}

	if resp.IsError() {
		if resp.StatusCode() == 404 {
			return fmt.Errorf("comment '%s' not found on issue '%s'", commentID, issueKey)
		}
		if resp.StatusCode() == 403 {
			return fmt.Errorf("you don't have permission to delete this comment")
		}
		return fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return nil
}
