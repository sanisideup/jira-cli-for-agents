package jira

import (
	"fmt"
	"sync"
	"time"

	"github.com/sanisideup/jira-cli-for-agents/pkg/client"
	"github.com/sanisideup/jira-cli-for-agents/pkg/models"
)

// MetadataService handles fetching and caching issue creation metadata
type MetadataService struct {
	client *client.Client
	cache  *metadataCache
}

// metadataCache stores create metadata with TTL
type metadataCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
}

// cacheEntry represents a cached metadata entry
type cacheEntry struct {
	data      *IssueTypeMeta
	expiresAt time.Time
}

// IssueTypeMeta represents metadata for a specific issue type in a project
type IssueTypeMeta struct {
	Name   string                    `json:"name"`
	Fields map[string]models.FieldMeta `json:"fields"`
}

const (
	// cacheTTL is the time-to-live for cached metadata (5 minutes)
	cacheTTL = 5 * time.Minute
)

// NewMetadataService creates a new MetadataService
func NewMetadataService(c *client.Client) *MetadataService {
	return &MetadataService{
		client: c,
		cache: &metadataCache{
			entries: make(map[string]*cacheEntry),
		},
	}
}

// GetCreateMetadata fetches create metadata for a specific project and issue type
// This returns information about what fields are available and required for issue creation
func (s *MetadataService) GetCreateMetadata(projectKey, issueType string) (*IssueTypeMeta, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", projectKey, issueType)
	if meta := s.cache.get(cacheKey); meta != nil {
		return meta, nil
	}

	// Fetch from API
	var response models.CreateMetaResponse

	resp, err := s.client.GetRequest().
		SetQueryParams(map[string]string{
			"projectKeys":     projectKey,
			"issuetypeNames":  issueType,
			"expand":          "projects.issuetypes.fields",
		}).
		SetResult(&response).
		Get("/issue/createmeta")

	if err != nil {
		return nil, fmt.Errorf("failed to fetch create metadata: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to fetch create metadata: HTTP %d", resp.StatusCode())
	}

	// Parse response
	if len(response.Projects) == 0 {
		return nil, fmt.Errorf("project '%s' not found or you don't have access", projectKey)
	}

	project := response.Projects[0]

	// Find the requested issue type
	if len(project.IssueTypes) == 0 {
		return nil, fmt.Errorf("issue type '%s' not found in project '%s'", issueType, projectKey)
	}

	// Use the first issue type (should match our filter)
	issueTypeData := project.IssueTypes[0]

	meta := &IssueTypeMeta{
		Name:   issueTypeData.Name,
		Fields: issueTypeData.Fields,
	}

	// Cache the result
	s.cache.set(cacheKey, meta)

	return meta, nil
}

// ValidateIssueData validates issue data against the metadata schema
// It checks for required fields, correct types, and allowed values
func (s *MetadataService) ValidateIssueData(projectKey, issueType string, data map[string]interface{}) error {
	meta, err := s.GetCreateMetadata(projectKey, issueType)
	if err != nil {
		return err
	}

	// Check all required fields are present
	for fieldID, fieldMeta := range meta.Fields {
		if !fieldMeta.Required {
			continue
		}

		// Skip 'reporter' field - it's automatically set to the current user
		// Even though the API says it's required, you cannot/should not provide it
		if fieldID == "reporter" {
			continue
		}

		value, exists := data[fieldID]
		if !exists {
			return fmt.Errorf("field '%s' (%s) is required for %s in project %s", fieldMeta.Name, fieldID, issueType, projectKey)
		}

		// Check for nil or empty values
		if value == nil {
			return fmt.Errorf("field '%s' (%s) cannot be nil", fieldMeta.Name, fieldID)
		}
	}

	// Validate field types and values
	for fieldID, value := range data {
		if value == nil {
			continue // Skip nil values for optional fields
		}

		fieldMeta, exists := meta.Fields[fieldID]
		if !exists {
			// Field not in metadata - this might be okay for some custom scenarios
			// We'll allow it but could add a warning mechanism later
			continue
		}

		// Validate type
		if err := s.validateFieldType(fieldID, fieldMeta, value); err != nil {
			return err
		}

		// Validate allowed values if applicable
		if len(fieldMeta.AllowedValues) > 0 {
			if err := s.validateAllowedValues(fieldID, fieldMeta, value); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateFieldType checks if the value matches the expected field type
func (s *MetadataService) validateFieldType(fieldID string, meta models.FieldMeta, value interface{}) error {
	schemaType := meta.Schema.Type

	switch schemaType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' (%s) expects string, got %T", meta.Name, fieldID, value)
		}
	case "number":
		switch value.(type) {
		case int, int64, float64, float32:
			// Valid number types
		default:
			return fmt.Errorf("field '%s' (%s) expects number, got %T", meta.Name, fieldID, value)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("field '%s' (%s) expects array, got %T", meta.Name, fieldID, value)
		}
	case "option", "priority", "user", "project", "issuetype":
		// These are typically objects with specific structures
		if _, ok := value.(map[string]interface{}); !ok {
			// Also allow string for some types (like project key)
			if _, ok := value.(string); !ok {
				return fmt.Errorf("field '%s' (%s) expects object or string, got %T", meta.Name, fieldID, value)
			}
		}
	}

	return nil
}

// validateAllowedValues checks if the value is in the list of allowed values
func (s *MetadataService) validateAllowedValues(fieldID string, meta models.FieldMeta, value interface{}) error {
	// For object values (like priority, status), we need to extract the name or id
	var valueToCheck string

	switch v := value.(type) {
	case string:
		valueToCheck = v
	case map[string]interface{}:
		// Try to get 'name' or 'id' field
		if name, ok := v["name"].(string); ok {
			valueToCheck = name
		} else if id, ok := v["id"].(string); ok {
			valueToCheck = id
		} else {
			// Can't validate object values without name or id
			return nil
		}
	default:
		// Can't validate other types
		return nil
	}

	// Check if value is in allowed values
	for _, allowed := range meta.AllowedValues {
		allowedMap, ok := allowed.(map[string]interface{})
		if !ok {
			continue
		}

		// Check both name and id
		if name, ok := allowedMap["name"].(string); ok && name == valueToCheck {
			return nil
		}
		if id, ok := allowedMap["id"].(string); ok && id == valueToCheck {
			return nil
		}
	}

	// Build list of allowed values for error message
	allowedList := make([]string, 0, len(meta.AllowedValues))
	for _, allowed := range meta.AllowedValues {
		if allowedMap, ok := allowed.(map[string]interface{}); ok {
			if name, ok := allowedMap["name"].(string); ok {
				allowedList = append(allowedList, name)
			}
		}
	}

	return fmt.Errorf("field '%s' (%s) value '%s' not in allowed values: %v", meta.Name, fieldID, valueToCheck, allowedList)
}

// get retrieves a cached entry if it exists and hasn't expired
func (c *metadataCache) get(key string) *IssueTypeMeta {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil
	}

	if time.Now().After(entry.expiresAt) {
		// Entry expired
		return nil
	}

	return entry.data
}

// set stores a new entry in the cache with TTL
func (c *metadataCache) set(key string, data *IssueTypeMeta) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(cacheTTL),
	}
}

// clear removes all cached entries
func (c *metadataCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry)
}
