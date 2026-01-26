package jira

import (
	"fmt"

	"github.com/sanisideup/jira-cli/pkg/client"
	"github.com/sanisideup/jira-cli/pkg/models"
)

// IssueService handles issue-related operations
type IssueService struct {
	client   *client.Client
	metadata *MetadataService
	fields   *FieldService
}

// CreateIssueRequest represents a request to create a single issue
type CreateIssueRequest struct {
	Fields map[string]interface{} `json:"fields"`
}

// BulkCreateRequest represents a request to create multiple issues
type BulkCreateRequest struct {
	IssueUpdates []CreateIssueRequest `json:"issueUpdates"`
}

// NewIssueService creates a new IssueService instance
func NewIssueService(c *client.Client) *IssueService {
	return &IssueService{
		client:   c,
		metadata: NewMetadataService(c),
		fields:   NewFieldService(c),
	}
}

// CreateIssue creates a single issue in Jira
// Returns the created issue's key, ID, and self URL
func (s *IssueService) CreateIssue(fields map[string]interface{}) (*models.IssueCreateResult, error) {
	// Prepare request
	req := CreateIssueRequest{
		Fields: fields,
	}

	var result models.IssueCreateResult
	var errorResp models.ErrorResponse

	// Make API request
	resp, err := s.client.PostRequest().
		SetBody(req).
		SetResult(&result).
		SetError(&errorResp).
		Post("/issue")

	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return &result, nil
}

// BulkCreateIssues creates multiple issues in a single request
// The Jira API supports up to 50 issues per request, so this method
// automatically chunks larger requests into batches
func (s *IssueService) BulkCreateIssues(issues []map[string]interface{}) (*models.BulkCreateResponse, error) {
	const maxBatchSize = 50

	// If we have 50 or fewer issues, make a single request
	if len(issues) <= maxBatchSize {
		return s.bulkCreateBatch(issues)
	}

	// Otherwise, chunk into multiple requests
	aggregatedResponse := &models.BulkCreateResponse{
		Issues: make([]models.IssueCreateResult, 0, len(issues)),
		Errors: make([]models.BulkCreateError, 0),
	}

	for i := 0; i < len(issues); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(issues) {
			end = len(issues)
		}

		batch := issues[i:end]
		resp, err := s.bulkCreateBatch(batch)
		if err != nil {
			return nil, fmt.Errorf("failed to create batch %d-%d: %w", i, end, err)
		}

		// Aggregate results
		aggregatedResponse.Issues = append(aggregatedResponse.Issues, resp.Issues...)

		// Adjust error indices to account for batching
		for _, bulkErr := range resp.Errors {
			bulkErr.FailedElementNumber += i
			aggregatedResponse.Errors = append(aggregatedResponse.Errors, bulkErr)
		}
	}

	return aggregatedResponse, nil
}

// bulkCreateBatch creates a single batch of issues (max 50)
func (s *IssueService) bulkCreateBatch(issues []map[string]interface{}) (*models.BulkCreateResponse, error) {
	// Prepare request
	issueUpdates := make([]CreateIssueRequest, len(issues))
	for i, fields := range issues {
		issueUpdates[i] = CreateIssueRequest{Fields: fields}
	}

	req := BulkCreateRequest{
		IssueUpdates: issueUpdates,
	}

	var result models.BulkCreateResponse
	var errorResp models.ErrorResponse

	// Make API request
	resp, err := s.client.PostRequest().
		SetBody(req).
		SetResult(&result).
		SetError(&errorResp).
		Post("/issue/bulk")

	if err != nil {
		return nil, fmt.Errorf("failed to bulk create issues: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return &result, nil
}

// GetIssue retrieves a single issue by its key or ID
func (s *IssueService) GetIssue(keyOrID string) (*models.Issue, error) {
	var issue models.Issue
	var errorResp models.ErrorResponse

	// Make API request
	resp, err := s.client.GetRequest().
		SetResult(&issue).
		SetError(&errorResp).
		Get(fmt.Sprintf("/issue/%s", keyOrID))

	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	if resp.IsError() {
		if resp.StatusCode() == 404 {
			return nil, fmt.Errorf("issue '%s' not found", keyOrID)
		}
		return nil, fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return &issue, nil
}

// ValidateIssueFields validates issue fields before creation
// This uses the metadata service to check required fields and types
func (s *IssueService) ValidateIssueFields(fields map[string]interface{}) error {
	// Extract project and issue type from fields
	project, ok := fields["project"].(map[string]interface{})
	if !ok {
		// Try string format (project key)
		projectStr, ok := fields["project"].(string)
		if !ok {
			return fmt.Errorf("field 'project' is required and must be an object or string")
		}
		project = map[string]interface{}{"key": projectStr}
	}

	projectKey, ok := project["key"].(string)
	if !ok {
		return fmt.Errorf("project must have a 'key' field")
	}

	issueType, ok := fields["issuetype"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("field 'issuetype' is required and must be an object")
	}

	issueTypeName, ok := issueType["name"].(string)
	if !ok {
		return fmt.Errorf("issuetype must have a 'name' field")
	}

	// Use metadata service to validate
	return s.metadata.ValidateIssueData(projectKey, issueTypeName, fields)
}
