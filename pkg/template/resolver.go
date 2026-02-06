package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TemplateResolver finds templates using a fallback chain of directories.
// Resolution order: explicit flag → local (.jcfa/templates) → config → user (~/.jcfa/templates) → builtin
type TemplateResolver struct {
	ExplicitDir string // --templates-dir flag (highest priority)
	LocalDir    string // ./.jcfa/templates/ (project-local)
	ConfigDir   string // from config.templates_dir
	UserDir     string // ~/.jcfa/templates/ (user default)
}

// TemplateInfo contains metadata about a resolved template
type TemplateInfo struct {
	Name   string `json:"name"`   // Template name (without .yaml extension)
	Path   string `json:"path"`   // Full path to the template file
	Source string `json:"source"` // Source: "explicit", "local", "config", "user", "builtin"
}

// NewResolver creates a new TemplateResolver with standard paths.
// configDir: from config.TemplatesDir (can be empty)
// explicitDir: from --templates-dir flag (can be empty)
func NewResolver(configDir, explicitDir string) *TemplateResolver {
	homeDir, _ := os.UserHomeDir()

	return &TemplateResolver{
		ExplicitDir: explicitDir,
		LocalDir:    ".jcfa/templates",
		ConfigDir:   configDir,
		UserDir:     filepath.Join(homeDir, ".jcfa", "templates"),
	}
}

// Resolve finds a template file using the fallback chain.
// Returns the full path to the template file and the source it was found in.
func (r *TemplateResolver) Resolve(name string) (path string, source string, err error) {
	filename := name + ".yaml"

	// Build search paths in priority order
	searchPaths := []struct {
		dir    string
		source string
	}{
		{r.ExplicitDir, "explicit"},
		{r.LocalDir, "local"},
		{r.ConfigDir, "config"},
		{r.UserDir, "user"},
	}

	var searchedPaths []string

	for _, sp := range searchPaths {
		if sp.dir == "" {
			continue
		}

		fullPath := filepath.Join(sp.dir, filename)
		searchedPaths = append(searchedPaths, fullPath)

		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, sp.source, nil
		}
	}

	return "", "", fmt.Errorf("template '%s' not found in:\n  %s", name, strings.Join(searchedPaths, "\n  "))
}

// ResolveWithBuiltin finds a template, including checking for builtin defaults.
// This is used by the Service which has access to embedded templates.
func (r *TemplateResolver) ResolveWithBuiltin(name string, builtinExists func(string) bool) (path string, source string, err error) {
	// First try filesystem paths
	path, source, err = r.Resolve(name)
	if err == nil {
		return path, source, nil
	}

	// Check if builtin exists
	if builtinExists(name) {
		return "", "builtin", nil
	}

	return "", "", err
}

// List returns all available templates from all directories, deduplicated by name.
// Earlier sources take priority (if a template exists in both local and user, local wins).
func (r *TemplateResolver) List() ([]TemplateInfo, error) {
	seen := make(map[string]bool)
	var templates []TemplateInfo

	dirs := []struct {
		path   string
		source string
	}{
		{r.ExplicitDir, "explicit"},
		{r.LocalDir, "local"},
		{r.ConfigDir, "config"},
		{r.UserDir, "user"},
	}

	for _, dir := range dirs {
		if dir.path == "" {
			continue
		}

		files, err := filepath.Glob(filepath.Join(dir.path, "*.yaml"))
		if err != nil {
			continue
		}

		for _, f := range files {
			name := strings.TrimSuffix(filepath.Base(f), ".yaml")
			if !seen[name] {
				seen[name] = true
				templates = append(templates, TemplateInfo{
					Name:   name,
					Path:   f,
					Source: dir.source,
				})
			}
		}
	}

	return templates, nil
}

// ListWithBuiltin returns all templates including builtin defaults.
// builtinNames should return a list of builtin template names.
func (r *TemplateResolver) ListWithBuiltin(builtinNames func() []string) ([]TemplateInfo, error) {
	templates, err := r.List()
	if err != nil {
		return nil, err
	}

	// Track what we've seen
	seen := make(map[string]bool)
	for _, t := range templates {
		seen[t.Name] = true
	}

	// Add builtins that haven't been overridden
	for _, name := range builtinNames() {
		if !seen[name] {
			templates = append(templates, TemplateInfo{
				Name:   name,
				Path:   "(builtin)",
				Source: "builtin",
			})
		}
	}

	return templates, nil
}

// GetLocalDir returns the project-local templates directory path
func (r *TemplateResolver) GetLocalDir() string {
	return r.LocalDir
}

// GetUserDir returns the user templates directory path
func (r *TemplateResolver) GetUserDir() string {
	return r.UserDir
}

// LocalDirExists checks if the project-local templates directory exists
func (r *TemplateResolver) LocalDirExists() bool {
	info, err := os.Stat(r.LocalDir)
	return err == nil && info.IsDir()
}
