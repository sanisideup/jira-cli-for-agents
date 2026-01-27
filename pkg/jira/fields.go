package jira

import (
	"fmt"
	"strings"

	"github.com/sanisideup/jira-cli-for-agents/pkg/client"
	"github.com/sanisideup/jira-cli-for-agents/pkg/config"
	"github.com/sanisideup/jira-cli-for-agents/pkg/models"
)

// FieldService handles field-related operations
type FieldService struct {
	client *client.Client
}

// NewFieldService creates a new FieldService instance
func NewFieldService(client *client.Client) *FieldService {
	return &FieldService{
		client: client,
	}
}

// ListFields retrieves all fields from Jira
// If projectKey is provided, it filters fields relevant to that project
func (s *FieldService) ListFields(projectKey string) ([]models.Field, error) {
	var fields []models.Field
	var errorResp models.ErrorResponse

	// Make request to get all fields
	resp, err := s.client.GetRequest().
		SetResult(&fields).
		SetError(&errorResp).
		Get("/field")

	if err != nil {
		return nil, fmt.Errorf("failed to fetch fields: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("API error: %s", formatErrorResponse(&errorResp))
	}

	// If project key is provided, we could filter results here
	// For now, return all fields as Jira's /field endpoint doesn't support project filtering
	// The filtering would need to be done via create metadata if needed in the future

	return fields, nil
}

// GetFieldByName searches for a field by name (case-insensitive)
func (s *FieldService) GetFieldByName(name string) (*models.Field, error) {
	fields, err := s.ListFields("")
	if err != nil {
		return nil, err
	}

	// Normalize search name
	searchName := strings.ToLower(strings.TrimSpace(name))

	// Search through fields
	for _, field := range fields {
		if strings.ToLower(field.Name) == searchName {
			return &field, nil
		}
	}

	return nil, fmt.Errorf("field '%s' not found", name)
}

// GetFieldByID retrieves a field by its ID
func (s *FieldService) GetFieldByID(id string) (*models.Field, error) {
	fields, err := s.ListFields("")
	if err != nil {
		return nil, err
	}

	for _, field := range fields {
		if field.ID == id {
			return &field, nil
		}
	}

	return nil, fmt.Errorf("field with ID '%s' not found", id)
}

// SaveFieldMapping saves a field alias to the config
// It validates that the field ID exists before saving
func (s *FieldService) SaveFieldMapping(alias, fieldID string, cfg *config.Config) error {
	// Validate that the field exists
	_, err := s.GetFieldByID(fieldID)
	if err != nil {
		return fmt.Errorf("cannot map alias '%s': %w", alias, err)
	}

	// Initialize field mappings if nil
	if cfg.FieldMappings == nil {
		cfg.FieldMappings = make(map[string]string)
	}

	// Check if alias already exists
	if existingID, exists := cfg.FieldMappings[alias]; exists {
		if existingID == fieldID {
			return fmt.Errorf("alias '%s' is already mapped to '%s'", alias, fieldID)
		}
		// Could add a --force flag in the future to allow overwriting
		return fmt.Errorf("alias '%s' already mapped to '%s'. Remove the existing mapping first", alias, existingID)
	}

	// Save the mapping
	cfg.FieldMappings[alias] = fieldID

	// Save config to disk
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save field mapping: %w", err)
	}

	return nil
}

// ResolveFieldID resolves an alias or field ID to the actual field ID
// It checks:
// 1. If the input is already a valid field ID (standard or custom)
// 2. If it's an alias in the config's field mappings
// 3. If it's a field name that can be resolved
func (s *FieldService) ResolveFieldID(nameOrID string, cfg *config.Config) (string, error) {
	nameOrID = strings.TrimSpace(nameOrID)

	// Check if it's already a field ID (customfield_* or standard field like "summary")
	if _, err := s.GetFieldByID(nameOrID); err == nil {
		return nameOrID, nil
	}

	// Check if it's an alias in field mappings
	if cfg.FieldMappings != nil {
		if fieldID, exists := cfg.FieldMappings[nameOrID]; exists {
			return fieldID, nil
		}
	}

	// Try to find by name
	field, err := s.GetFieldByName(nameOrID)
	if err != nil {
		return "", fmt.Errorf("could not resolve '%s': not a valid field ID, alias, or field name. Run 'jcfa fields list' to see available fields", nameOrID)
	}

	return field.ID, nil
}

// formatErrorResponse formats a Jira error response for display
func formatErrorResponse(errResp *models.ErrorResponse) string {
	var messages []string

	if len(errResp.ErrorMessages) > 0 {
		messages = append(messages, strings.Join(errResp.ErrorMessages, "; "))
	}

	if len(errResp.Errors) > 0 {
		for field, msg := range errResp.Errors {
			messages = append(messages, fmt.Sprintf("%s: %s", field, msg))
		}
	}

	if len(messages) == 0 {
		return "unknown error"
	}

	return strings.Join(messages, "; ")
}
