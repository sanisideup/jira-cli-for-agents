package jira

import (
	"fmt"
	"strings"

	"github.com/sanisideup/jira-cli-for-agents/pkg/client"
	"github.com/sanisideup/jira-cli-for-agents/pkg/config"
	"github.com/sanisideup/jira-cli-for-agents/pkg/models"
)

// LinkService handles issue linking operations, including epic-story relationships
type LinkService struct {
	client *client.Client
	fields *FieldService
}

// IssueLinkRequest represents a request to create a link between two issues
type IssueLinkRequest struct {
	Type         IssueLinkType `json:"type"`
	InwardIssue  IssueRef      `json:"inwardIssue"`
	OutwardIssue IssueRef      `json:"outwardIssue"`
}

// IssueLinkType represents the type of link to create
type IssueLinkType struct {
	Name string `json:"name"`
}

// IssueRef represents a reference to an issue
type IssueRef struct {
	Key string `json:"key"`
}

// NewLinkService creates a new LinkService instance
func NewLinkService(c *client.Client) *LinkService {
	return &LinkService{
		client: c,
		fields: NewFieldService(c),
	}
}

// LinkToEpic links a story to an epic using the appropriate method for the Jira instance
// It tries two strategies:
// 1. Update the Epic Link custom field (older Jira instances)
// 2. Create an issue link with "Epic-Story" relationship (newer Jira instances)
func (s *LinkService) LinkToEpic(storyKey, epicKey string, cfg *config.Config) error {
	// Strategy 1: Try updating Epic Link field
	epicLinkField, err := s.DetectEpicLinkField(cfg)
	if err == nil && epicLinkField != "" {
		// Update the epic link field
		return s.updateEpicLinkField(storyKey, epicKey, epicLinkField)
	}

	// Strategy 2: Try creating an issue link
	return s.createEpicStoryLink(storyKey, epicKey)
}

// DetectEpicLinkField detects the Epic Link custom field ID
// It checks common IDs and searches for fields with "Epic Link" in the name
func (s *LinkService) DetectEpicLinkField(cfg *config.Config) (string, error) {
	// Check if we already have it mapped in config
	if cfg.FieldMappings != nil {
		if epicLinkID, exists := cfg.FieldMappings["epic_link"]; exists {
			return epicLinkID, nil
		}
	}

	// Common Epic Link field IDs to try
	commonIDs := []string{
		"customfield_10014", // Most common
		"customfield_10008", // Alternative common ID
		"customfield_10011", // Another alternative
	}

	// Try to find the field
	fields, err := s.fields.ListFields("")
	if err != nil {
		return "", err
	}

	// Check common IDs first
	for _, commonID := range commonIDs {
		for _, field := range fields {
			if field.ID == commonID && strings.Contains(strings.ToLower(field.Name), "epic") {
				// Found it! Save to config for future use
				if cfg.FieldMappings == nil {
					cfg.FieldMappings = make(map[string]string)
				}
				cfg.FieldMappings["epic_link"] = field.ID
				_ = cfg.Save() // Best effort save
				return field.ID, nil
			}
		}
	}

	// Search by name
	for _, field := range fields {
		fieldNameLower := strings.ToLower(field.Name)
		if strings.Contains(fieldNameLower, "epic") && strings.Contains(fieldNameLower, "link") {
			// Found it! Save to config for future use
			if cfg.FieldMappings == nil {
				cfg.FieldMappings = make(map[string]string)
			}
			cfg.FieldMappings["epic_link"] = field.ID
			_ = cfg.Save() // Best effort save
			return field.ID, nil
		}
	}

	return "", fmt.Errorf("epic link field not found")
}

// updateEpicLinkField updates the Epic Link custom field for a story
func (s *LinkService) updateEpicLinkField(storyKey, epicKey, epicLinkField string) error {
	// Prepare update request
	updateReq := map[string]interface{}{
		"fields": map[string]interface{}{
			epicLinkField: epicKey,
		},
	}

	var errorResp models.ErrorResponse

	// Make API request
	resp, err := s.client.PutRequest().
		SetBody(updateReq).
		SetError(&errorResp).
		Put(fmt.Sprintf("/issue/%s", storyKey))

	if err != nil {
		return fmt.Errorf("failed to update epic link: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("API error updating epic link: %s", formatErrorResponse(&errorResp))
	}

	return nil
}

// createEpicStoryLink creates an issue link between an epic and a story
func (s *LinkService) createEpicStoryLink(storyKey, epicKey string) error {
	// Prepare link request
	// The epic is the "inward" issue and the story is the "outward" issue
	linkReq := IssueLinkRequest{
		Type: IssueLinkType{
			Name: "Epic-Story Link", // This is a common link type name
		},
		InwardIssue: IssueRef{
			Key: epicKey,
		},
		OutwardIssue: IssueRef{
			Key: storyKey,
		},
	}

	var errorResp models.ErrorResponse

	// Make API request
	resp, err := s.client.PostRequest().
		SetBody(linkReq).
		SetError(&errorResp).
		Post("/issueLink")

	if err != nil {
		return fmt.Errorf("failed to create epic-story link: %w", err)
	}

	if resp.IsError() {
		// Try alternative link type names
		alternativeTypes := []string{"Relates", "Blocks", "Dependency"}
		for _, linkType := range alternativeTypes {
			linkReq.Type.Name = linkType
			resp, err = s.client.PostRequest().
				SetBody(linkReq).
				SetError(&errorResp).
				Post("/issueLink")

			if err == nil && !resp.IsError() {
				return nil
			}
		}

		return fmt.Errorf("API error creating epic-story link: %s", formatErrorResponse(&errorResp))
	}

	return nil
}

// CreateIssueLink creates a link between two issues with a specified link type
func (s *LinkService) CreateIssueLink(inwardKey, outwardKey, linkTypeName string) error {
	linkReq := IssueLinkRequest{
		Type: IssueLinkType{
			Name: linkTypeName,
		},
		InwardIssue: IssueRef{
			Key: inwardKey,
		},
		OutwardIssue: IssueRef{
			Key: outwardKey,
		},
	}

	var errorResp models.ErrorResponse

	resp, err := s.client.PostRequest().
		SetBody(linkReq).
		SetError(&errorResp).
		Post("/issueLink")

	if err != nil {
		return fmt.Errorf("failed to create issue link: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return nil
}

// GetAvailableLinkTypes retrieves all available issue link types from the Jira instance.
// Returns a slice of link types with their inward/outward descriptions.
func (s *LinkService) GetAvailableLinkTypes() ([]models.IssueLinkType, error) {
	var linkTypesResp models.IssueLinkTypeResponse
	var errorResp models.ErrorResponse

	resp, err := s.client.GetRequest().
		SetResult(&linkTypesResp).
		SetError(&errorResp).
		Get("/issueLinkType")

	if err != nil {
		return nil, fmt.Errorf("failed to get link types: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return linkTypesResp.IssueLinkTypes, nil
}

// GetIssueLinks retrieves all links for a specific issue.
// It fetches the issue with only the issuelinks field to minimize data transfer.
// Parameters:
//   - issueKey: The issue key (e.g., "PROJ-123")
//
// Returns a slice of IssueLink objects containing link details and related issues.
func (s *LinkService) GetIssueLinks(issueKey string) ([]models.IssueLink, error) {
	if issueKey == "" {
		return nil, fmt.Errorf("issue key cannot be empty")
	}

	// Fetch issue with only issuelinks field to minimize response size
	var issue models.Issue
	var errorResp models.ErrorResponse

	resp, err := s.client.GetRequest().
		SetQueryParam("fields", "issuelinks").
		SetResult(&issue).
		SetError(&errorResp).
		Get(fmt.Sprintf("/issue/%s", issueKey))

	if err != nil {
		return nil, fmt.Errorf("failed to get issue links for %s: %w", issueKey, err)
	}

	if resp.IsError() {
		if resp.StatusCode() == 404 {
			return nil, fmt.Errorf("issue '%s' not found", issueKey)
		}
		return nil, fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	// Extract issuelinks from the fields map
	issueLinks := extractIssueLinks(issue.Fields)

	return issueLinks, nil
}

// extractIssueLinks extracts and parses issue links from the issue fields map.
// Handles the dynamic JSON structure returned by Jira API.
// Note: Returns empty slice (not error) for missing/invalid data to be lenient with API responses.
func extractIssueLinks(fields map[string]interface{}) []models.IssueLink {
	linksRaw, ok := fields["issuelinks"]
	if !ok || linksRaw == nil {
		return []models.IssueLink{}
	}

	linksArray, ok := linksRaw.([]interface{})
	if !ok {
		return []models.IssueLink{}
	}

	var links []models.IssueLink
	for _, linkRaw := range linksArray {
		linkMap, ok := linkRaw.(map[string]interface{})
		if !ok {
			continue
		}

		link := models.IssueLink{}

		// Extract link ID
		if id, ok := linkMap["id"].(string); ok {
			link.ID = id
		}

		// Extract self URL
		if self, ok := linkMap["self"].(string); ok {
			link.Self = self
		}

		// Extract link type
		if typeMap, ok := linkMap["type"].(map[string]interface{}); ok {
			link.Type = models.IssueLinkType{
				ID:      getStringField(typeMap, "id"),
				Name:    getStringField(typeMap, "name"),
				Inward:  getStringField(typeMap, "inward"),
				Outward: getStringField(typeMap, "outward"),
				Self:    getStringField(typeMap, "self"),
			}
		}

		// Extract outward issue (if present)
		if outwardMap, ok := linkMap["outwardIssue"].(map[string]interface{}); ok {
			link.OutwardIssue = parseIssueRef(outwardMap)
		}

		// Extract inward issue (if present)
		if inwardMap, ok := linkMap["inwardIssue"].(map[string]interface{}); ok {
			link.InwardIssue = parseIssueRef(inwardMap)
		}

		links = append(links, link)
	}

	return links
}

// parseIssueRef parses a map into an IssueRef struct
func parseIssueRef(m map[string]interface{}) *models.IssueRef {
	ref := &models.IssueRef{
		ID:   getStringField(m, "id"),
		Key:  getStringField(m, "key"),
		Self: getStringField(m, "self"),
	}

	// Extract nested fields if present
	if fieldsMap, ok := m["fields"].(map[string]interface{}); ok {
		ref.Fields = fieldsMap
	}

	return ref
}

// getStringField safely extracts a string field from a map
func getStringField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// DeleteIssueLink deletes an issue link by its ID.
// The link ID can be obtained from GetIssueLinks.
// Parameters:
//   - linkID: The unique identifier of the link to delete
//
// Returns an error if the deletion fails or the link doesn't exist.
func (s *LinkService) DeleteIssueLink(linkID string) error {
	if linkID == "" {
		return fmt.Errorf("link ID cannot be empty")
	}

	var errorResp models.ErrorResponse

	resp, err := s.client.DeleteRequest().
		SetError(&errorResp).
		Delete(fmt.Sprintf("/issueLink/%s", linkID))

	if err != nil {
		return fmt.Errorf("failed to delete link %s: %w", linkID, err)
	}

	if resp.IsError() {
		if resp.StatusCode() == 404 {
			return fmt.Errorf("link ID '%s' not found", linkID)
		}
		if resp.StatusCode() == 403 {
			return fmt.Errorf("you don't have permission to delete this link")
		}
		return fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	return nil
}
