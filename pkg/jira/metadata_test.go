package jira

import (
	"testing"

	"github.com/sanisideup/jira-cli-for-agents/pkg/models"
)

func TestMetadataCache(t *testing.T) {
	cache := &metadataCache{
		entries: make(map[string]*cacheEntry),
	}

	// Test cache miss
	if result := cache.get("test:key"); result != nil {
		t.Error("Expected cache miss for non-existent key")
	}

	// Test cache set and get
	meta := &IssueTypeMeta{
		Name: "Story",
		Fields: map[string]models.FieldMeta{
			"summary": {
				Required: true,
				Name:     "Summary",
				Schema: models.FieldSchema{
					Type: "string",
				},
			},
		},
	}

	cache.set("test:key", meta)

	result := cache.get("test:key")
	if result == nil {
		t.Fatal("Expected cache hit")
	}

	if result.Name != "Story" {
		t.Errorf("Expected name 'Story', got '%s'", result.Name)
	}

	if len(result.Fields) != 1 {
		t.Errorf("Expected 1 field, got %d", len(result.Fields))
	}
}

func TestValidateFieldType(t *testing.T) {
	svc := &MetadataService{}

	tests := []struct {
		name      string
		fieldMeta models.FieldMeta
		value     interface{}
		expectErr bool
	}{
		{
			name: "valid string",
			fieldMeta: models.FieldMeta{
				Name:   "Summary",
				Schema: models.FieldSchema{Type: "string"},
			},
			value:     "Test summary",
			expectErr: false,
		},
		{
			name: "invalid string",
			fieldMeta: models.FieldMeta{
				Name:   "Summary",
				Schema: models.FieldSchema{Type: "string"},
			},
			value:     123,
			expectErr: true,
		},
		{
			name: "valid number - int",
			fieldMeta: models.FieldMeta{
				Name:   "Story Points",
				Schema: models.FieldSchema{Type: "number"},
			},
			value:     5,
			expectErr: false,
		},
		{
			name: "valid number - float",
			fieldMeta: models.FieldMeta{
				Name:   "Story Points",
				Schema: models.FieldSchema{Type: "number"},
			},
			value:     5.5,
			expectErr: false,
		},
		{
			name: "invalid number",
			fieldMeta: models.FieldMeta{
				Name:   "Story Points",
				Schema: models.FieldSchema{Type: "number"},
			},
			value:     "not a number",
			expectErr: true,
		},
		{
			name: "valid array",
			fieldMeta: models.FieldMeta{
				Name:   "Labels",
				Schema: models.FieldSchema{Type: "array"},
			},
			value:     []interface{}{"label1", "label2"},
			expectErr: false,
		},
		{
			name: "invalid array",
			fieldMeta: models.FieldMeta{
				Name:   "Labels",
				Schema: models.FieldSchema{Type: "array"},
			},
			value:     "not an array",
			expectErr: true,
		},
		{
			name: "valid object - map",
			fieldMeta: models.FieldMeta{
				Name:   "Priority",
				Schema: models.FieldSchema{Type: "priority"},
			},
			value:     map[string]interface{}{"name": "High"},
			expectErr: false,
		},
		{
			name: "valid object - string",
			fieldMeta: models.FieldMeta{
				Name:   "Project",
				Schema: models.FieldSchema{Type: "project"},
			},
			value:     "PROJ",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateFieldType("testfield", tt.fieldMeta, tt.value)
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateAllowedValues(t *testing.T) {
	svc := &MetadataService{}

	allowedValues := []interface{}{
		map[string]interface{}{"id": "1", "name": "High"},
		map[string]interface{}{"id": "2", "name": "Medium"},
		map[string]interface{}{"id": "3", "name": "Low"},
	}

	fieldMeta := models.FieldMeta{
		Name:          "Priority",
		AllowedValues: allowedValues,
	}

	tests := []struct {
		name      string
		value     interface{}
		expectErr bool
	}{
		{
			name:      "valid name",
			value:     map[string]interface{}{"name": "High"},
			expectErr: false,
		},
		{
			name:      "valid id",
			value:     map[string]interface{}{"id": "2"},
			expectErr: false,
		},
		{
			name:      "valid string name",
			value:     "Low",
			expectErr: false,
		},
		{
			name:      "invalid value",
			value:     map[string]interface{}{"name": "Critical"},
			expectErr: true,
		},
		{
			name:      "invalid string",
			value:     "VeryHigh",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateAllowedValues("priority", fieldMeta, tt.value)
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestCacheClear(t *testing.T) {
	cache := &metadataCache{
		entries: make(map[string]*cacheEntry),
	}

	// Add some entries
	meta := &IssueTypeMeta{Name: "Story"}
	cache.set("key1", meta)
	cache.set("key2", meta)

	if len(cache.entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(cache.entries))
	}

	// Clear cache
	cache.clear()

	if len(cache.entries) != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", len(cache.entries))
	}

	// Verify cache miss after clear
	if result := cache.get("key1"); result != nil {
		t.Error("Expected cache miss after clear")
	}
}
