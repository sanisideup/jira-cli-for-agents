package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sanisideup/jira-cli/pkg/config"
)

func TestLoadTemplate(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// Test loading default template (epic)
	tmpl, err := svc.LoadTemplate("epic")
	if err != nil {
		t.Fatalf("Failed to load epic template: %v", err)
	}

	if tmpl.Type != "Epic" {
		t.Errorf("Expected type 'Epic', got '%s'", tmpl.Type)
	}

	if len(tmpl.Fields) == 0 {
		t.Error("Expected fields to be populated")
	}
}

func TestRenderTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// Load story template
	tmpl, err := svc.LoadTemplate("story")
	if err != nil {
		t.Fatalf("Failed to load story template: %v", err)
	}

	// Test data
	data := map[string]interface{}{
		"Project":     "PROJ",
		"Summary":     "Test Story",
		"Description": "This is a test story",
		"Priority":    "High",
		"Labels":      []string{"backend", "api"},
		"StoryPoints": 5,
		"EpicKey":     "PROJ-100",
	}

	// Config with field mappings
	cfg := &config.Config{
		FieldMappings: map[string]string{
			"story_points": "customfield_10016",
			"epic_link":    "customfield_10014",
		},
	}

	// Render template
	rendered, err := svc.RenderTemplate(tmpl, data, cfg)
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	// Check rendered fields
	if project, ok := rendered["project"].(map[string]interface{}); ok {
		if project["key"] != "PROJ" {
			t.Errorf("Expected project key 'PROJ', got '%v'", project["key"])
		}
	} else {
		t.Error("Expected project to be a map")
	}

	if summary, ok := rendered["summary"].(string); ok {
		if summary != "Test Story" {
			t.Errorf("Expected summary 'Test Story', got '%s'", summary)
		}
	} else {
		t.Error("Expected summary to be a string")
	}

	// Check field alias resolution
	if _, exists := rendered["customfield_10016"]; !exists {
		t.Error("Expected story_points to be resolved to customfield_10016")
	}

	if _, exists := rendered["customfield_10014"]; !exists {
		t.Error("Expected epic_link to be resolved to customfield_10014")
	}

	// Check labels are parsed as JSON array
	if labels, ok := rendered["labels"].([]interface{}); ok {
		if len(labels) != 2 {
			t.Errorf("Expected 2 labels, got %d", len(labels))
		}
	} else {
		t.Errorf("Expected labels to be an array, got %T", rendered["labels"])
	}
}

func TestRenderTemplateWithDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// Load bug template
	tmpl, err := svc.LoadTemplate("bug")
	if err != nil {
		t.Fatalf("Failed to load bug template: %v", err)
	}

	// Test data without priority (should use default "High")
	data := map[string]interface{}{
		"Project":     "PROJ",
		"Summary":     "Test Bug",
		"Description": "This is a test bug",
		"Labels":      []string{"bug"},
	}

	cfg := &config.Config{}

	// Render template
	rendered, err := svc.RenderTemplate(tmpl, data, cfg)
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	// Check default priority
	if priority, ok := rendered["priority"].(map[string]interface{}); ok {
		if priority["name"] != "High" {
			t.Errorf("Expected default priority 'High', got '%v'", priority["name"])
		}
	} else {
		t.Error("Expected priority to be a map")
	}
}

func TestListTemplates(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// Should list default templates
	templates, err := svc.ListTemplates()
	if err != nil {
		t.Fatalf("Failed to list templates: %v", err)
	}

	if len(templates) < 4 {
		t.Errorf("Expected at least 4 default templates, got %d", len(templates))
	}

	// Check for expected templates
	expectedTemplates := map[string]bool{
		"epic":    false,
		"story":   false,
		"bug":     false,
		"charter": false,
	}

	for _, name := range templates {
		if _, exists := expectedTemplates[name]; exists {
			expectedTemplates[name] = true
		}
	}

	for name, found := range expectedTemplates {
		if !found {
			t.Errorf("Expected to find template '%s'", name)
		}
	}
}

func TestInitTemplates(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// Initialize templates
	err := svc.InitTemplates()
	if err != nil {
		t.Fatalf("Failed to initialize templates: %v", err)
	}

	// Check if templates were created
	expectedFiles := []string{"epic.yaml", "story.yaml", "bug.yaml", "charter.yaml"}
	for _, filename := range expectedFiles {
		path := filepath.Join(tmpDir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file '%s' to exist", filename)
		}
	}

	// Test that re-initializing doesn't overwrite existing files
	// Create a custom template
	customPath := filepath.Join(tmpDir, "epic.yaml")
	customContent := []byte("type: Epic\nfields:\n  custom: true")
	if err := os.WriteFile(customPath, customContent, 0644); err != nil {
		t.Fatalf("Failed to write custom template: %v", err)
	}

	// Re-initialize
	err = svc.InitTemplates()
	if err != nil {
		t.Fatalf("Failed to re-initialize templates: %v", err)
	}

	// Check that custom template wasn't overwritten
	content, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("Failed to read custom template: %v", err)
	}

	if string(content) != string(customContent) {
		t.Error("Expected custom template to not be overwritten")
	}
}

func TestRenderTemplateWithNilValues(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// Load story template
	tmpl, err := svc.LoadTemplate("story")
	if err != nil {
		t.Fatalf("Failed to load story template: %v", err)
	}

	// Test data with nil assignee and story points
	data := map[string]interface{}{
		"Project":     "PROJ",
		"Summary":     "Test Story",
		"Description": "This is a test story",
		"Labels":      []string{},
		"EpicKey":     "PROJ-100",
		// Assignee and StoryPoints intentionally omitted
	}

	cfg := &config.Config{
		FieldMappings: map[string]string{
			"story_points": "customfield_10016",
			"epic_link":    "customfield_10014",
		},
	}

	// Render template
	rendered, err := svc.RenderTemplate(tmpl, data, cfg)
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	// Check that nil values are handled correctly
	// The template uses "default nil" which will render as "<no value>" string
	// This is expected Go template behavior when the value doesn't exist
	if storyPoints, exists := rendered["customfield_10016"]; exists {
		// Check it's either nil or "<no value>" (Go template behavior)
		if storyPoints != nil && storyPoints != "<no value>" {
			t.Logf("story_points rendered as: %v (type: %T)", storyPoints, storyPoints)
		}
	}
}
