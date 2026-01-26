package jira

import (
	"fmt"
	"strings"

	"github.com/sanisideup/jira-cli/pkg/client"
	"github.com/sanisideup/jira-cli/pkg/models"
)

// SearchService handles issue search operations
type SearchService struct {
	client *client.Client
}

// NewSearchService creates a new search service
func NewSearchService(client *client.Client) *SearchService {
	return &SearchService{client: client}
}

// SearchRequest represents a JQL search request
type SearchRequest struct {
	JQL        string   `json:"jql"`
	StartAt    int      `json:"startAt,omitempty"`
	MaxResults int      `json:"maxResults,omitempty"`
	Fields     []string `json:"fields,omitempty"`
}

// Search executes a JQL query and returns matching issues
// Parameters:
//   - jql: JQL query string (e.g., "project = PROJ AND status = Open")
//   - maxResults: Maximum number of results to return (0 = default 50)
//   - fields: List of fields to include in response (nil = all fields)
func (s *SearchService) Search(jql string, maxResults int, fields []string) (*models.SearchResponse, error) {
	if jql == "" {
		return nil, fmt.Errorf("JQL query cannot be empty")
	}

	// Default to 50 results if not specified
	if maxResults <= 0 {
		maxResults = 50
	}

	// If no specific fields requested, get common fields for display
	if fields == nil || len(fields) == 0 {
		fields = []string{
			"summary",
			"status",
			"issuetype",
			"assignee",
			"priority",
			"created",
			"updated",
			"description",
			"labels",
		}
	}

	req := SearchRequest{
		JQL:        jql,
		MaxResults: maxResults,
		Fields:     fields,
	}

	var result models.SearchResponse
	var errorResp models.ErrorResponse

	resp, err := s.client.HTTPClient.R().
		SetBody(req).
		SetResult(&result).
		SetError(&errorResp).
		Post("/search/jql")

	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return &result, nil
}

// GetIssue retrieves a single issue by key or ID
// Returns full issue details including all fields
func (s *SearchService) GetIssue(keyOrID string) (*models.Issue, error) {
	if keyOrID == "" {
		return nil, fmt.Errorf("issue key or ID cannot be empty")
	}

	var issue models.Issue
	var errorResp models.ErrorResponse

	resp, err := s.client.HTTPClient.R().
		SetResult(&issue).
		SetError(&errorResp).
		Get(fmt.Sprintf("/issue/%s", keyOrID))

	if err != nil {
		return nil, fmt.Errorf("failed to get issue %s: %w", keyOrID, err)
	}

	if resp.IsError() {
		if resp.StatusCode() == 404 {
			return nil, fmt.Errorf("issue '%s' not found", keyOrID)
		}
		return nil, fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return &issue, nil
}

// UpdateIssue updates fields on an existing issue
// Parameters:
//   - keyOrID: Issue key (e.g., "PROJ-123") or ID
//   - fields: Map of field IDs to values to update
func (s *SearchService) UpdateIssue(keyOrID string, fields map[string]interface{}) error {
	if keyOrID == "" {
		return fmt.Errorf("issue key or ID cannot be empty")
	}

	if len(fields) == 0 {
		return fmt.Errorf("no fields to update")
	}

	body := map[string]interface{}{
		"fields": fields,
	}

	var errorResp models.ErrorResponse

	resp, err := s.client.HTTPClient.R().
		SetBody(body).
		SetError(&errorResp).
		Put(fmt.Sprintf("/issue/%s", keyOrID))

	if err != nil {
		return fmt.Errorf("failed to update issue %s: %w", keyOrID, err)
	}

	if resp.IsError() {
		return fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return nil
}

// AddComment adds a comment to an issue
func (s *SearchService) AddComment(keyOrID, text string) (*models.Comment, error) {
	if keyOrID == "" {
		return nil, fmt.Errorf("issue key or ID cannot be empty")
	}

	if text == "" {
		return nil, fmt.Errorf("comment text cannot be empty")
	}

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
		Post(fmt.Sprintf("/issue/%s/comment", keyOrID))

	if err != nil {
		return nil, fmt.Errorf("failed to add comment to %s: %w", keyOrID, err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return &comment, nil
}

// GetTransitions retrieves available transitions for an issue
func (s *SearchService) GetTransitions(keyOrID string) ([]models.Transition, error) {
	if keyOrID == "" {
		return nil, fmt.Errorf("issue key or ID cannot be empty")
	}

	var result models.TransitionsResponse
	var errorResp models.ErrorResponse

	resp, err := s.client.HTTPClient.R().
		SetResult(&result).
		SetError(&errorResp).
		Get(fmt.Sprintf("/issue/%s/transitions", keyOrID))

	if err != nil {
		return nil, fmt.Errorf("failed to get transitions for %s: %w", keyOrID, err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return result.Transitions, nil
}

// TransitionIssue transitions an issue to a new status
// Parameters:
//   - keyOrID: Issue key or ID
//   - statusName: Target status name (case-insensitive)
func (s *SearchService) TransitionIssue(keyOrID, statusName string) error {
	if keyOrID == "" {
		return fmt.Errorf("issue key or ID cannot be empty")
	}

	if statusName == "" {
		return fmt.Errorf("status name cannot be empty")
	}

	// Get available transitions
	transitions, err := s.GetTransitions(keyOrID)
	if err != nil {
		return err
	}

	// Find matching transition (case-insensitive)
	var transitionID string
	for _, t := range transitions {
		if strings.EqualFold(t.To.Name, statusName) {
			transitionID = t.ID
			break
		}
	}

	if transitionID == "" {
		available := make([]string, len(transitions))
		for i, t := range transitions {
			available[i] = t.To.Name
		}
		return fmt.Errorf("status '%s' not found. Available transitions: %v", statusName, available)
	}

	// Execute transition
	body := map[string]interface{}{
		"transition": map[string]interface{}{
			"id": transitionID,
		},
	}

	var errorResp models.ErrorResponse

	resp, err := s.client.HTTPClient.R().
		SetBody(body).
		SetError(&errorResp).
		Post(fmt.Sprintf("/issue/%s/transitions", keyOrID))

	if err != nil {
		return fmt.Errorf("failed to transition issue %s: %w", keyOrID, err)
	}

	if resp.IsError() {
		return fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return nil
}

// LinkIssues creates a link between two issues
// Parameters:
//   - inwardKey: Key of the inward issue
//   - outwardKey: Key of the outward issue
//   - linkType: Type of link (e.g., "Blocks", "Relates", "Duplicate")
func (s *SearchService) LinkIssues(inwardKey, outwardKey, linkType string) error {
	if inwardKey == "" || outwardKey == "" {
		return fmt.Errorf("both issue keys are required")
	}

	if linkType == "" {
		return fmt.Errorf("link type cannot be empty")
	}

	body := map[string]interface{}{
		"type": map[string]interface{}{
			"name": linkType,
		},
		"inwardIssue": map[string]interface{}{
			"key": inwardKey,
		},
		"outwardIssue": map[string]interface{}{
			"key": outwardKey,
		},
	}

	var errorResp models.ErrorResponse

	resp, err := s.client.HTTPClient.R().
		SetBody(body).
		SetError(&errorResp).
		Post("/issueLink")

	if err != nil {
		return fmt.Errorf("failed to link issues: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return nil
}
