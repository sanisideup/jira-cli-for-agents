package template

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sanisideup/jira-cli-for-agents/pkg/config"
	"gopkg.in/yaml.v3"
)

//go:embed defaults/*.yaml
var defaultTemplates embed.FS

// Service handles template loading, rendering, and management
type Service struct {
	templateDir string
}

// Template represents a Jira issue template
type Template struct {
	Type   string                 `yaml:"type"`   // Issue type name (e.g., "Story", "Epic", "Bug")
	Fields map[string]interface{} `yaml:"fields"` // Field values with {{ }} placeholders
}

// NewService creates a new template service
func NewService(templateDir string) *Service {
	return &Service{
		templateDir: templateDir,
	}
}

// LoadTemplate loads a template by name from the user's template directory or embedded defaults
func (s *Service) LoadTemplate(name string) (*Template, error) {
	// Try user template directory first
	userTemplatePath := filepath.Join(s.templateDir, name+".yaml")
	if data, err := os.ReadFile(userTemplatePath); err == nil {
		return s.parseTemplate(data)
	}

	// Fallback to embedded default templates
	defaultPath := fmt.Sprintf("defaults/%s.yaml", name)
	data, err := defaultTemplates.ReadFile(defaultPath)
	if err != nil {
		return nil, fmt.Errorf("template '%s' not found in %s or defaults", name, s.templateDir)
	}

	return s.parseTemplate(data)
}

// parseTemplate parses YAML template data into a Template struct
func (s *Service) parseTemplate(data []byte) (*Template, error) {
	var tmpl Template
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	if tmpl.Type == "" {
		return nil, fmt.Errorf("template missing required 'type' field")
	}

	return &tmpl, nil
}

// RenderTemplate renders a template with the provided data and resolves field aliases
func (s *Service) RenderTemplate(tmpl *Template, data map[string]interface{}, cfg *config.Config) (map[string]interface{}, error) {
	// Create a copy of template fields
	renderedFields := make(map[string]interface{})

	// Custom template functions
	funcMap := template.FuncMap{
		"toJson": func(v interface{}) string {
			if v == nil {
				return "null"
			}
			b, err := json.Marshal(v)
			if err != nil {
				return "null"
			}
			return string(b)
		},
		"default": func(defaultValue, value interface{}) interface{} {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},
	}

	// Render each field value
	for fieldKey, fieldValue := range tmpl.Fields {
		rendered, err := s.renderValue(fieldValue, data, funcMap)
		if err != nil {
			return nil, fmt.Errorf("failed to render field '%s': %w", fieldKey, err)
		}

		// Resolve field aliases to actual field IDs
		actualFieldID := s.resolveFieldID(fieldKey, cfg)
		renderedFields[actualFieldID] = rendered
	}

	return renderedFields, nil
}

// renderValue recursively renders a value, handling strings, maps, arrays, and primitives
func (s *Service) renderValue(value interface{}, data map[string]interface{}, funcMap template.FuncMap) (interface{}, error) {
	switch v := value.(type) {
	case string:
		// Render string templates
		tmpl, err := template.New("field").Funcs(funcMap).Parse(v)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template: %w", err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("failed to execute template: %w", err)
		}

		result := buf.String()

		// Try to parse as JSON if it looks like JSON
		if strings.HasPrefix(strings.TrimSpace(result), "[") || strings.HasPrefix(strings.TrimSpace(result), "{") {
			var parsed interface{}
			if err := json.Unmarshal([]byte(result), &parsed); err == nil {
				return parsed, nil
			}
		}

		// Special handling for "null" string
		if result == "null" {
			return nil, nil
		}

		return result, nil

	case map[string]interface{}:
		// Recursively render map values
		rendered := make(map[string]interface{})
		for k, val := range v {
			renderedVal, err := s.renderValue(val, data, funcMap)
			if err != nil {
				return nil, err
			}
			rendered[k] = renderedVal
		}
		return rendered, nil

	case []interface{}:
		// Recursively render array values
		rendered := make([]interface{}, len(v))
		for i, val := range v {
			renderedVal, err := s.renderValue(val, data, funcMap)
			if err != nil {
				return nil, err
			}
			rendered[i] = renderedVal
		}
		return rendered, nil

	default:
		// Return primitives as-is (int, float, bool, nil)
		return v, nil
	}
}

// resolveFieldID resolves a field alias to its actual field ID using config mappings
func (s *Service) resolveFieldID(fieldKey string, cfg *config.Config) string {
	// Check if there's a mapping for this alias
	if cfg != nil && cfg.FieldMappings != nil {
		if actualID, exists := cfg.FieldMappings[fieldKey]; exists {
			return actualID
		}
	}

	// Return as-is if no mapping found (might already be a field ID)
	return fieldKey
}

// ListTemplates returns a list of available template names
func (s *Service) ListTemplates() ([]string, error) {
	templates := make(map[string]bool)

	// List user templates
	if entries, err := os.ReadDir(s.templateDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
				name := strings.TrimSuffix(entry.Name(), ".yaml")
				templates[name] = true
			}
		}
	}

	// List default templates
	if entries, err := defaultTemplates.ReadDir("defaults"); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
				name := strings.TrimSuffix(entry.Name(), ".yaml")
				templates[name] = true
			}
		}
	}

	// Convert to sorted list
	result := make([]string, 0, len(templates))
	for name := range templates {
		result = append(result, name)
	}

	return result, nil
}

// InitTemplates copies default templates to the user's template directory
func (s *Service) InitTemplates() error {
	// Create template directory if it doesn't exist
	if err := os.MkdirAll(s.templateDir, 0755); err != nil {
		return fmt.Errorf("failed to create template directory: %w", err)
	}

	// Read all default templates
	entries, err := defaultTemplates.ReadDir("defaults")
	if err != nil {
		return fmt.Errorf("failed to read default templates: %w", err)
	}

	// Copy each default template
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		// Read default template
		data, err := defaultTemplates.ReadFile(filepath.Join("defaults", entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to read default template %s: %w", entry.Name(), err)
		}

		// Write to user template directory
		userPath := filepath.Join(s.templateDir, entry.Name())

		// Don't overwrite existing templates
		if _, err := os.Stat(userPath); err == nil {
			continue
		}

		if err := os.WriteFile(userPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write template %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// GetTemplateDir returns the template directory path
func (s *Service) GetTemplateDir() string {
	return s.templateDir
}
