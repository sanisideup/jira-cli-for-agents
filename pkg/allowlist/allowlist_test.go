// Package allowlist provides command restriction tests for sandboxed/agent execution.
package allowlist

import (
	"os"
	"sort"
	"strings"
	"testing"
)

// Helper function to clear environment variables before each test
func clearEnvVars(t *testing.T) {
	t.Helper()
	os.Unsetenv(EnvReadOnly)
	os.Unsetenv(EnvCommandAllowlist)
}

// Helper function to set environment variables with cleanup
func setEnv(t *testing.T, key, value string) {
	t.Helper()
	os.Setenv(key, value)
	t.Cleanup(func() {
		os.Unsetenv(key)
	})
}

// =============================================================================
// TestNewChecker - Initialization Tests
// =============================================================================

func TestNewChecker_NoEnvVars(t *testing.T) {
	clearEnvVars(t)

	c := NewChecker()

	if c.enabled {
		t.Error("expected enabled=false when no env vars are set")
	}
	if c.readOnly {
		t.Error("expected readOnly=false when no env vars are set")
	}
	if len(c.allowedCommands) != 0 {
		t.Errorf("expected empty allowedCommands, got %d", len(c.allowedCommands))
	}
}

func TestNewChecker_ReadOnlyMode(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvReadOnly, "1")

	c := NewChecker()

	if !c.enabled {
		t.Error("expected enabled=true in read-only mode")
	}
	if !c.readOnly {
		t.Error("expected readOnly=true when JIRA_READONLY is set")
	}
	// Should have all read-only commands
	for _, cmd := range ReadOnlyCommands {
		if !c.allowedCommands[cmd] {
			t.Errorf("expected command %q to be allowed in read-only mode", cmd)
		}
	}
}

func TestNewChecker_ReadOnlyWithTrueString(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvReadOnly, "true")

	c := NewChecker()

	if !c.enabled {
		t.Error("expected enabled=true when JIRA_READONLY='true'")
	}
	if !c.readOnly {
		t.Error("expected readOnly=true when JIRA_READONLY='true'")
	}
}

func TestNewChecker_ExplicitAllowlist(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvCommandAllowlist, "get,search,list")

	c := NewChecker()

	if !c.enabled {
		t.Error("expected enabled=true with explicit allowlist")
	}
	if c.readOnly {
		t.Error("expected readOnly=false with explicit allowlist (not JIRA_READONLY)")
	}

	expected := map[string]bool{"get": true, "search": true, "list": true}
	for cmd := range expected {
		if !c.allowedCommands[cmd] {
			t.Errorf("expected command %q to be in allowlist", cmd)
		}
	}
	if len(c.allowedCommands) != len(expected) {
		t.Errorf("expected %d commands in allowlist, got %d", len(expected), len(c.allowedCommands))
	}
}

func TestNewChecker_BothEnvVarsSet_ReadOnlyWins(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvReadOnly, "1")
	setEnv(t, EnvCommandAllowlist, "get,create")

	c := NewChecker()

	if !c.readOnly {
		t.Error("expected readOnly=true when both env vars are set (readonly takes precedence)")
	}
	// Should have read-only commands, not the explicit allowlist
	if c.allowedCommands["create"] {
		t.Error("expected 'create' to NOT be allowed when read-only mode takes precedence")
	}
}

func TestNewChecker_EmptyAllowlistString(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvCommandAllowlist, "")

	c := NewChecker()

	if c.enabled {
		t.Error("expected enabled=false when allowlist is empty string")
	}
}

func TestNewChecker_WhitespaceInAllowlist(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvCommandAllowlist, " get , search , list ")

	c := NewChecker()

	if !c.enabled {
		t.Error("expected enabled=true")
	}

	// Commands should be trimmed
	for _, cmd := range []string{"get", "search", "list"} {
		if !c.allowedCommands[cmd] {
			t.Errorf("expected trimmed command %q to be allowed", cmd)
		}
	}
	// Verify no entries with whitespace
	for cmd := range c.allowedCommands {
		if cmd != strings.TrimSpace(cmd) {
			t.Errorf("command %q was not trimmed properly", cmd)
		}
	}
}

func TestNewChecker_CaseInsensitiveAllowlist(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvCommandAllowlist, "GET,Search,LIST")

	c := NewChecker()

	// All should be converted to lowercase
	for _, cmd := range []string{"get", "search", "list"} {
		if !c.allowedCommands[cmd] {
			t.Errorf("expected lowercase command %q to be in allowlist", cmd)
		}
	}
}

// =============================================================================
// TestIsAllowed - Permission Checks
// =============================================================================

func TestIsAllowed_DisabledChecker(t *testing.T) {
	clearEnvVars(t)

	c := NewChecker()

	// When disabled, all commands should be allowed
	testCmds := []string{"get", "create", "delete", "anything", ""}
	for _, cmd := range testCmds {
		if !c.IsAllowed(cmd) {
			t.Errorf("expected %q to be allowed when checker is disabled", cmd)
		}
	}
}

func TestIsAllowed_ReadOnlyMode(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvReadOnly, "1")

	c := NewChecker()

	testCases := []struct {
		command  string
		expected bool
	}{
		// Read commands should be allowed
		{"get", true},
		{"search", true},
		{"list", true},
		{"fields", true},
		{"comments list", true},
		{"comments get", true},
		{"link list", true},
		{"link types", true},
		{"attachment list", true},

		// Write commands should be blocked
		{"create", false},
		{"update", false},
		{"delete", false},
		{"transition", false},
		{"comment", false},
		{"comments add", false},
		{"comments delete", false},
		{"batch create", false},
		{"link create", false},
		{"link delete", false},
		{"attachment upload", false},
		{"attachment delete", false},
		{"configure", false},
	}

	for _, tc := range testCases {
		t.Run(tc.command, func(t *testing.T) {
			result := c.IsAllowed(tc.command)
			if result != tc.expected {
				t.Errorf("IsAllowed(%q) = %v, expected %v", tc.command, result, tc.expected)
			}
		})
	}
}

func TestIsAllowed_ExplicitAllowlist(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvCommandAllowlist, "get,search")

	c := NewChecker()

	testCases := []struct {
		command  string
		expected bool
	}{
		{"get", true},
		{"search", true},
		{"list", false},   // Not in allowlist
		{"create", false}, // Not in allowlist
	}

	for _, tc := range testCases {
		t.Run(tc.command, func(t *testing.T) {
			result := c.IsAllowed(tc.command)
			if result != tc.expected {
				t.Errorf("IsAllowed(%q) = %v, expected %v", tc.command, result, tc.expected)
			}
		})
	}
}

func TestIsAllowed_CaseInsensitivity(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvCommandAllowlist, "get,search")

	c := NewChecker()

	testCases := []struct {
		command  string
		expected bool
	}{
		{"get", true},
		{"GET", true},
		{"Get", true},
		{"gET", true},
		{"SEARCH", true},
		{"Search", true},
	}

	for _, tc := range testCases {
		t.Run(tc.command, func(t *testing.T) {
			result := c.IsAllowed(tc.command)
			if result != tc.expected {
				t.Errorf("IsAllowed(%q) = %v, expected %v", tc.command, result, tc.expected)
			}
		})
	}
}

func TestIsAllowed_CommandWithWhitespace(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvCommandAllowlist, "get,search")

	c := NewChecker()

	testCases := []struct {
		command  string
		expected bool
	}{
		{"  get  ", true},
		{"  search", true},
		{"get  ", true},
	}

	for _, tc := range testCases {
		t.Run(tc.command, func(t *testing.T) {
			result := c.IsAllowed(tc.command)
			if result != tc.expected {
				t.Errorf("IsAllowed(%q) = %v, expected %v", tc.command, result, tc.expected)
			}
		})
	}
}

// =============================================================================
// TestAlwaysAllowedCommands - Bypass Commands
// =============================================================================

func TestIsAllowed_AlwaysAllowedCommands(t *testing.T) {
	clearEnvVars(t)
	// Even with a restrictive allowlist, these should work
	setEnv(t, EnvCommandAllowlist, "get")

	c := NewChecker()

	alwaysAllowed := []string{"help", "version", "--help", "-h"}
	for _, cmd := range alwaysAllowed {
		t.Run(cmd, func(t *testing.T) {
			if !c.IsAllowed(cmd) {
				t.Errorf("expected %q to always be allowed", cmd)
			}
		})
	}
}

func TestIsAllowed_AlwaysAllowedInReadOnlyMode(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvReadOnly, "1")

	c := NewChecker()

	alwaysAllowed := []string{"help", "version", "--help", "-h"}
	for _, cmd := range alwaysAllowed {
		t.Run(cmd, func(t *testing.T) {
			if !c.IsAllowed(cmd) {
				t.Errorf("expected %q to always be allowed even in read-only mode", cmd)
			}
		})
	}
}

// =============================================================================
// TestCheck - Error Message Validation
// =============================================================================

func TestCheck_AllowedCommand(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvReadOnly, "1")

	c := NewChecker()

	err := c.Check("get")
	if err != nil {
		t.Errorf("expected no error for allowed command, got: %v", err)
	}
}

func TestCheck_BlockedInReadOnlyMode(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvReadOnly, "1")

	c := NewChecker()

	err := c.Check("create")
	if err == nil {
		t.Fatal("expected error for blocked command in read-only mode")
	}

	if !strings.Contains(err.Error(), "JIRA_READONLY mode enabled") {
		t.Errorf("error message should mention JIRA_READONLY, got: %v", err)
	}
	if !strings.Contains(err.Error(), "create") {
		t.Errorf("error message should mention the blocked command 'create', got: %v", err)
	}
}

func TestCheck_BlockedByExplicitAllowlist(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvCommandAllowlist, "get,search")

	c := NewChecker()

	err := c.Check("delete")
	if err == nil {
		t.Fatal("expected error for command not in allowlist")
	}

	if !strings.Contains(err.Error(), "not in the allowlist") {
		t.Errorf("error message should mention 'not in the allowlist', got: %v", err)
	}
	if !strings.Contains(err.Error(), EnvCommandAllowlist) {
		t.Errorf("error message should mention env var name, got: %v", err)
	}
}

func TestCheck_DisabledChecker(t *testing.T) {
	clearEnvVars(t)

	c := NewChecker()

	// All commands should pass when disabled
	for _, cmd := range []string{"get", "create", "delete", "anything"} {
		err := c.Check(cmd)
		if err != nil {
			t.Errorf("expected no error when disabled, got: %v for command %q", err, cmd)
		}
	}
}

// =============================================================================
// TestGetAllowedCommands - List Retrieval
// =============================================================================

func TestGetAllowedCommands_ReadOnlyMode(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvReadOnly, "1")

	c := NewChecker()

	allowed := c.GetAllowedCommands()
	if allowed == nil {
		t.Fatal("expected non-nil slice for read-only mode")
	}

	if len(allowed) != len(ReadOnlyCommands) {
		t.Errorf("expected %d commands, got %d", len(ReadOnlyCommands), len(allowed))
	}

	// Verify all read-only commands are present
	allowedMap := make(map[string]bool)
	for _, cmd := range allowed {
		allowedMap[cmd] = true
	}
	for _, cmd := range ReadOnlyCommands {
		if !allowedMap[cmd] {
			t.Errorf("expected %q to be in allowed commands", cmd)
		}
	}
}

func TestGetAllowedCommands_ExplicitAllowlist(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvCommandAllowlist, "get,search,list")

	c := NewChecker()

	allowed := c.GetAllowedCommands()
	if len(allowed) != 3 {
		t.Errorf("expected 3 commands, got %d", len(allowed))
	}

	// Sort for consistent comparison
	sort.Strings(allowed)
	expected := []string{"get", "list", "search"}
	for i, cmd := range expected {
		if allowed[i] != cmd {
			t.Errorf("expected %q at index %d, got %q", cmd, i, allowed[i])
		}
	}
}

func TestGetAllowedCommands_Disabled(t *testing.T) {
	clearEnvVars(t)

	c := NewChecker()

	allowed := c.GetAllowedCommands()
	if allowed != nil {
		t.Errorf("expected nil when disabled, got: %v", allowed)
	}
}

// =============================================================================
// TestAllCommands - Command Registry
// =============================================================================

func TestAllCommands_ReturnsAllCommands(t *testing.T) {
	all := AllCommands()

	expectedCount := len(ReadOnlyCommands) + len(WriteCommands)
	if len(all) != expectedCount {
		t.Errorf("expected %d commands, got %d", expectedCount, len(all))
	}
}

func TestAllCommands_ContainsReadOnlyCommands(t *testing.T) {
	all := AllCommands()
	allMap := make(map[string]bool)
	for _, cmd := range all {
		allMap[cmd] = true
	}

	for _, cmd := range ReadOnlyCommands {
		if !allMap[cmd] {
			t.Errorf("expected ReadOnlyCommand %q to be in AllCommands()", cmd)
		}
	}
}

func TestAllCommands_ContainsWriteCommands(t *testing.T) {
	all := AllCommands()
	allMap := make(map[string]bool)
	for _, cmd := range all {
		allMap[cmd] = true
	}

	for _, cmd := range WriteCommands {
		if !allMap[cmd] {
			t.Errorf("expected WriteCommand %q to be in AllCommands()", cmd)
		}
	}
}

func TestAllCommands_NoDuplicates(t *testing.T) {
	all := AllCommands()
	seen := make(map[string]bool)

	for _, cmd := range all {
		if seen[cmd] {
			t.Errorf("duplicate command found: %q", cmd)
		}
		seen[cmd] = true
	}
}

// =============================================================================
// TestIsEnabled and TestIsReadOnly - State Accessors
// =============================================================================

func TestIsEnabled(t *testing.T) {
	testCases := []struct {
		name        string
		envReadOnly string
		envAllowlist string
		expected    bool
	}{
		{"no env vars", "", "", false},
		{"read-only mode", "1", "", true},
		{"explicit allowlist", "", "get,search", true},
		{"both set", "1", "get", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clearEnvVars(t)
			if tc.envReadOnly != "" {
				setEnv(t, EnvReadOnly, tc.envReadOnly)
			}
			if tc.envAllowlist != "" {
				setEnv(t, EnvCommandAllowlist, tc.envAllowlist)
			}

			c := NewChecker()
			if c.IsEnabled() != tc.expected {
				t.Errorf("IsEnabled() = %v, expected %v", c.IsEnabled(), tc.expected)
			}
		})
	}
}

func TestIsReadOnly(t *testing.T) {
	testCases := []struct {
		name        string
		envReadOnly string
		envAllowlist string
		expected    bool
	}{
		{"no env vars", "", "", false},
		{"read-only mode", "1", "", true},
		{"explicit allowlist only", "", "get,search", false},
		{"both set (readonly wins)", "1", "get", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clearEnvVars(t)
			if tc.envReadOnly != "" {
				setEnv(t, EnvReadOnly, tc.envReadOnly)
			}
			if tc.envAllowlist != "" {
				setEnv(t, EnvCommandAllowlist, tc.envAllowlist)
			}

			c := NewChecker()
			if c.IsReadOnly() != tc.expected {
				t.Errorf("IsReadOnly() = %v, expected %v", c.IsReadOnly(), tc.expected)
			}
		})
	}
}

// =============================================================================
// TestEdgeCases - Edge Case Tests
// =============================================================================

func TestEdgeCases_EmptyCommand(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvCommandAllowlist, "get,search")

	c := NewChecker()

	// Empty string should be blocked (not in allowlist)
	if c.IsAllowed("") {
		t.Error("expected empty command to be blocked")
	}
}

func TestEdgeCases_PartialCommandMatch(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvCommandAllowlist, "get,search")

	c := NewChecker()

	// Partial matches should NOT work - exact match required
	partials := []string{"ge", "g", "searc", "getx", "xget"}
	for _, cmd := range partials {
		if c.IsAllowed(cmd) {
			t.Errorf("expected partial match %q to be blocked", cmd)
		}
	}
}

func TestEdgeCases_UnknownCommand(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvReadOnly, "1")

	c := NewChecker()

	unknown := []string{"foobar", "unknown", "notacommand", "xyz123"}
	for _, cmd := range unknown {
		if c.IsAllowed(cmd) {
			t.Errorf("expected unknown command %q to be blocked", cmd)
		}
	}
}

func TestEdgeCases_CommandWithSpecialChars(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvCommandAllowlist, "get")

	c := NewChecker()

	// Commands with special characters should be blocked
	special := []string{"get;rm", "get&delete", "get|cat", "get$(whoami)"}
	for _, cmd := range special {
		if c.IsAllowed(cmd) {
			t.Errorf("expected command with special chars %q to be blocked", cmd)
		}
	}
}

func TestEdgeCases_OnlyWhitespaceCommand(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvCommandAllowlist, "get")

	c := NewChecker()

	// After trimming, these become empty strings
	whitespaceOnly := []string{"   ", "\t", "\n", " \t\n "}
	for _, cmd := range whitespaceOnly {
		if c.IsAllowed(cmd) {
			t.Errorf("expected whitespace-only command %q to be blocked", cmd)
		}
	}
}

func TestEdgeCases_AllowlistWithEmptyEntries(t *testing.T) {
	clearEnvVars(t)
	// Allowlist with empty entries due to double commas
	setEnv(t, EnvCommandAllowlist, "get,,search,,,list")

	c := NewChecker()

	// Should still work correctly
	if !c.IsAllowed("get") {
		t.Error("expected 'get' to be allowed")
	}
	if !c.IsAllowed("search") {
		t.Error("expected 'search' to be allowed")
	}
	if !c.IsAllowed("list") {
		t.Error("expected 'list' to be allowed")
	}

	// Empty string should not be in allowlist
	if c.allowedCommands[""] {
		t.Error("empty string should not be added to allowedCommands")
	}
}

// =============================================================================
// TestNestedCommands - Subcommand Tests
// =============================================================================

func TestNestedCommands_ReadOnlyMode(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvReadOnly, "1")

	c := NewChecker()

	testCases := []struct {
		command  string
		expected bool
	}{
		// Allowed nested commands
		{"comments list", true},
		{"comments get", true},
		{"link list", true},
		{"link types", true},
		{"attachment list", true},

		// Blocked nested commands
		{"comments add", false},
		{"comments update", false},
		{"comments delete", false},
		{"link create", false},
		{"link delete", false},
		{"attachment upload", false},
		{"attachment delete", false},
		{"batch create", false},
	}

	for _, tc := range testCases {
		t.Run(tc.command, func(t *testing.T) {
			result := c.IsAllowed(tc.command)
			if result != tc.expected {
				t.Errorf("IsAllowed(%q) = %v, expected %v", tc.command, result, tc.expected)
			}
		})
	}
}

func TestNestedCommands_ExplicitAllowlist(t *testing.T) {
	clearEnvVars(t)
	setEnv(t, EnvCommandAllowlist, "comments list,link create")

	c := NewChecker()

	testCases := []struct {
		command  string
		expected bool
	}{
		{"comments list", true},
		{"link create", true},
		{"comments add", false},
		{"link delete", false},
		{"get", false}, // Not in explicit list
	}

	for _, tc := range testCases {
		t.Run(tc.command, func(t *testing.T) {
			result := c.IsAllowed(tc.command)
			if result != tc.expected {
				t.Errorf("IsAllowed(%q) = %v, expected %v", tc.command, result, tc.expected)
			}
		})
	}
}

// =============================================================================
// TestReadOnlyAndWriteCommandLists - Verify Command Lists
// =============================================================================

func TestReadOnlyCommandsAreValid(t *testing.T) {
	// Ensure ReadOnlyCommands contains expected commands
	expected := map[string]bool{
		"get":             true,
		"search":          true,
		"list":            true,
		"fields":          true,
		"version":         true,
		"help":            true,
		"attachment list": true,
		"comments list":   true,
		"comments get":    true,
		"link list":       true,
		"link types":      true,
	}

	for _, cmd := range ReadOnlyCommands {
		if !expected[cmd] {
			t.Errorf("unexpected command in ReadOnlyCommands: %q", cmd)
		}
	}

	if len(ReadOnlyCommands) != len(expected) {
		t.Errorf("ReadOnlyCommands count mismatch: got %d, expected %d", len(ReadOnlyCommands), len(expected))
	}
}

func TestWriteCommandsAreValid(t *testing.T) {
	// Ensure WriteCommands contains expected commands
	expected := map[string]bool{
		"create":            true,
		"update":            true,
		"transition":        true,
		"comment":           true,
		"comments add":      true,
		"comments update":   true,
		"comments delete":   true,
		"batch":             true,
		"batch create":      true,
		"link":              true,
		"link create":       true,
		"link delete":       true,
		"attachment upload": true,
		"attachment delete": true,
		"configure":         true,
		"template":          true,
	}

	for _, cmd := range WriteCommands {
		if !expected[cmd] {
			t.Errorf("unexpected command in WriteCommands: %q", cmd)
		}
	}

	if len(WriteCommands) != len(expected) {
		t.Errorf("WriteCommands count mismatch: got %d, expected %d", len(WriteCommands), len(expected))
	}
}

func TestNoOverlapBetweenReadAndWriteCommands(t *testing.T) {
	readSet := make(map[string]bool)
	for _, cmd := range ReadOnlyCommands {
		readSet[cmd] = true
	}

	for _, cmd := range WriteCommands {
		if readSet[cmd] {
			t.Errorf("command %q appears in both ReadOnlyCommands and WriteCommands", cmd)
		}
	}
}
