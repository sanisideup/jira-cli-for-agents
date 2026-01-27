// Package allowlist provides command restriction for sandboxed/agent execution.
package allowlist

import (
	"fmt"
	"os"
	"strings"
)

const (
	// EnvCommandAllowlist is the environment variable for allowed commands
	EnvCommandAllowlist = "JIRA_COMMAND_ALLOWLIST"

	// EnvReadOnly restricts to read-only commands only
	EnvReadOnly = "JIRA_READONLY"
)

// ReadOnlyCommands are commands that don't modify data
var ReadOnlyCommands = []string{
	"get",
	"search",
	"list",
	"fields",
	"version",
	"help",
	"attachment list",
	"comments list",
	"comments get",
	"link list",
	"link types",
}

// WriteCommands are commands that modify data
var WriteCommands = []string{
	"create",
	"update",
	"transition",
	"comment",
	"comments add",
	"comments update",
	"comments delete",
	"batch",
	"batch create",
	"link",
	"link create",
	"link delete",
	"attachment upload",
	"attachment delete",
	"configure",
	"template",
}

// Checker validates commands against the allowlist
type Checker struct {
	allowedCommands map[string]bool
	readOnly        bool
	enabled         bool
}

// NewChecker creates a new allowlist checker from environment variables
func NewChecker() *Checker {
	c := &Checker{
		allowedCommands: make(map[string]bool),
	}

	// Check for read-only mode
	if os.Getenv(EnvReadOnly) != "" {
		c.readOnly = true
		c.enabled = true
		for _, cmd := range ReadOnlyCommands {
			c.allowedCommands[cmd] = true
		}
		return c
	}

	// Check for explicit allowlist
	allowlist := os.Getenv(EnvCommandAllowlist)
	if allowlist == "" {
		c.enabled = false
		return c
	}

	c.enabled = true
	commands := strings.Split(allowlist, ",")
	for _, cmd := range commands {
		cmd = strings.TrimSpace(strings.ToLower(cmd))
		if cmd != "" {
			c.allowedCommands[cmd] = true
		}
	}

	return c
}

// IsAllowed checks if a command is allowed to run
func (c *Checker) IsAllowed(command string) bool {
	if !c.enabled {
		return true // No restrictions
	}

	command = strings.ToLower(strings.TrimSpace(command))

	// Always allow help and version
	if command == "help" || command == "version" || command == "--help" || command == "-h" {
		return true
	}

	return c.allowedCommands[command]
}

// Check validates a command and returns an error if not allowed
func (c *Checker) Check(command string) error {
	if !c.IsAllowed(command) {
		if c.readOnly {
			return fmt.Errorf("command '%s' is blocked: JIRA_READONLY mode enabled (only read operations allowed)", command)
		}
		return fmt.Errorf("command '%s' is not in the allowlist (set via %s)", command, EnvCommandAllowlist)
	}
	return nil
}

// IsEnabled returns whether the allowlist is active
func (c *Checker) IsEnabled() bool {
	return c.enabled
}

// IsReadOnly returns whether read-only mode is active
func (c *Checker) IsReadOnly() bool {
	return c.readOnly
}

// GetAllowedCommands returns the list of allowed commands
func (c *Checker) GetAllowedCommands() []string {
	if !c.enabled {
		return nil
	}

	commands := make([]string, 0, len(c.allowedCommands))
	for cmd := range c.allowedCommands {
		commands = append(commands, cmd)
	}
	return commands
}

// AllCommands returns all available commands (for documentation)
func AllCommands() []string {
	all := make([]string, 0, len(ReadOnlyCommands)+len(WriteCommands))
	all = append(all, ReadOnlyCommands...)
	all = append(all, WriteCommands...)
	return all
}
