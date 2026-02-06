package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolver_Resolve(t *testing.T) {
	// Create temp directories for testing
	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, "local")
	userDir := filepath.Join(tmpDir, "user")
	explicitDir := filepath.Join(tmpDir, "explicit")

	// Create directories
	os.MkdirAll(localDir, 0755)
	os.MkdirAll(userDir, 0755)
	os.MkdirAll(explicitDir, 0755)

	// Create test templates
	os.WriteFile(filepath.Join(localDir, "local-only.yaml"), []byte("type: LocalOnly"), 0644)
	os.WriteFile(filepath.Join(localDir, "both.yaml"), []byte("type: LocalBoth"), 0644)
	os.WriteFile(filepath.Join(userDir, "user-only.yaml"), []byte("type: UserOnly"), 0644)
	os.WriteFile(filepath.Join(userDir, "both.yaml"), []byte("type: UserBoth"), 0644)
	os.WriteFile(filepath.Join(explicitDir, "explicit-only.yaml"), []byte("type: ExplicitOnly"), 0644)

	tests := []struct {
		name        string
		template    string
		explicitDir string
		wantSource  string
		wantErr     bool
	}{
		{
			name:       "local only template found in local",
			template:   "local-only",
			wantSource: "local",
		},
		{
			name:       "user only template found in user",
			template:   "user-only",
			wantSource: "user",
		},
		{
			name:       "local takes priority over user",
			template:   "both",
			wantSource: "local",
		},
		{
			name:        "explicit takes priority over all",
			template:    "explicit-only",
			explicitDir: explicitDir,
			wantSource:  "explicit",
		},
		{
			name:     "nonexistent template returns error",
			template: "nonexistent",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &TemplateResolver{
				ExplicitDir: tt.explicitDir,
				LocalDir:    localDir,
				ConfigDir:   "",
				UserDir:     userDir,
			}

			path, source, err := resolver.Resolve(tt.template)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Resolve() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Resolve() unexpected error: %v", err)
				return
			}

			if source != tt.wantSource {
				t.Errorf("Resolve() source = %v, want %v", source, tt.wantSource)
			}

			if path == "" {
				t.Errorf("Resolve() path is empty")
			}
		})
	}
}

func TestResolver_List(t *testing.T) {
	// Create temp directories for testing
	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, "local")
	userDir := filepath.Join(tmpDir, "user")

	// Create directories
	os.MkdirAll(localDir, 0755)
	os.MkdirAll(userDir, 0755)

	// Create test templates
	os.WriteFile(filepath.Join(localDir, "epic.yaml"), []byte("type: Epic"), 0644)
	os.WriteFile(filepath.Join(localDir, "story.yaml"), []byte("type: Story"), 0644)
	os.WriteFile(filepath.Join(userDir, "story.yaml"), []byte("type: StoryUser"), 0644)
	os.WriteFile(filepath.Join(userDir, "bug.yaml"), []byte("type: Bug"), 0644)

	resolver := &TemplateResolver{
		LocalDir: localDir,
		UserDir:  userDir,
	}

	templates, err := resolver.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	// Should have 3 unique templates: epic (local), story (local - wins over user), bug (user)
	if len(templates) != 3 {
		t.Errorf("List() returned %d templates, want 3", len(templates))
	}

	// Verify sources
	sources := make(map[string]string)
	for _, tmpl := range templates {
		sources[tmpl.Name] = tmpl.Source
	}

	if sources["epic"] != "local" {
		t.Errorf("epic source = %v, want local", sources["epic"])
	}
	if sources["story"] != "local" {
		t.Errorf("story source = %v, want local (should override user)", sources["story"])
	}
	if sources["bug"] != "user" {
		t.Errorf("bug source = %v, want user", sources["bug"])
	}
}

func TestResolver_EmptyDirs(t *testing.T) {
	resolver := &TemplateResolver{
		LocalDir: "",
		UserDir:  "",
	}

	_, _, err := resolver.Resolve("anything")
	if err == nil {
		t.Error("Resolve() with empty dirs should return error")
	}

	templates, err := resolver.List()
	if err != nil {
		t.Errorf("List() with empty dirs should not error: %v", err)
	}
	if len(templates) != 0 {
		t.Errorf("List() with empty dirs should return empty list, got %d", len(templates))
	}
}
