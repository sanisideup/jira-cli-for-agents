package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sanisideup/jira-cli-for-agents/pkg/allowlist"
	"github.com/spf13/cobra"
)

// allowlistCmd is the parent command for allowlist operations
var allowlistCmd = &cobra.Command{
	Use:   "allowlist",
	Short: "Manage command allowlist restrictions",
	Long: `Manage command allowlist restrictions for secure CLI usage.

The allowlist feature restricts which commands can be executed, useful for:
- AI agents that should only read data (read-only mode)
- Sandboxed environments with limited permissions
- Security-conscious automation scripts

Configuration is done via environment variables:
  JIRA_READONLY=1              Enable read-only mode (blocks all write commands)
  JIRA_COMMAND_ALLOWLIST=...   Allow only specific commands (comma-separated)

Subcommands:
  status    - Show current allowlist status
  commands  - List all commands by category (read/write)
  check     - Check if a specific command is allowed
  enable    - Show instructions to enable allowlist`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// allowlistStatusCmd shows the current allowlist status
var allowlistStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current allowlist status",
	Long: `Show the current status of the command allowlist.

Displays:
- Whether the allowlist is enabled
- Which mode is active (read-only or explicit allowlist)
- List of currently allowed commands

Examples:
  jcfa allowlist status
  jcfa allowlist status --json`,
	RunE: runAllowlistStatus,
}

// allowlistCommandsCmd lists all commands by category
var allowlistCommandsCmd = &cobra.Command{
	Use:   "commands",
	Short: "List all commands by category",
	Long: `List all available CLI commands organized by category (read/write).

This helps you understand which commands are safe for read-only operations
and which ones modify data.

Examples:
  jcfa allowlist commands
  jcfa allowlist commands --json`,
	RunE: runAllowlistCommands,
}

// allowlistCheckCmd checks if a specific command is allowed
var allowlistCheckCmd = &cobra.Command{
	Use:   "check <command>",
	Short: "Check if a command is allowed",
	Long: `Check if a specific command is allowed under current restrictions.

Returns exit code 0 if allowed, 1 if blocked.

Examples:
  jcfa allowlist check get
  jcfa allowlist check create
  jcfa allowlist check "comments list"`,
	Args: cobra.ExactArgs(1),
	RunE: runAllowlistCheck,
}

// allowlistEnableCmd shows instructions to enable allowlist
var allowlistEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Show instructions to enable allowlist",
	Long: `Show instructions for enabling the command allowlist.

Provides copy-paste commands for:
- Enabling read-only mode
- Setting up a custom allowlist
- Making settings permanent`,
	Run: runAllowlistEnable,
}

func init() {
	// Add subcommands to allowlist
	allowlistCmd.AddCommand(allowlistStatusCmd)
	allowlistCmd.AddCommand(allowlistCommandsCmd)
	allowlistCmd.AddCommand(allowlistCheckCmd)
	allowlistCmd.AddCommand(allowlistEnableCmd)

	// Register allowlist command
	rootCmd.AddCommand(allowlistCmd)
}

func runAllowlistStatus(cmd *cobra.Command, args []string) error {
	checker := allowlist.NewChecker()

	if jsonOutput {
		status := map[string]interface{}{
			"enabled":         checker.IsEnabled(),
			"readOnly":        checker.IsReadOnly(),
			"allowedCommands": checker.GetAllowedCommands(),
			"envVars": map[string]string{
				"JIRA_READONLY":          os.Getenv(allowlist.EnvReadOnly),
				"JIRA_COMMAND_ALLOWLIST": os.Getenv(allowlist.EnvCommandAllowlist),
			},
		}
		return outputJSON(status)
	}

	fmt.Println("Command Allowlist Status")
	fmt.Println("========================")
	fmt.Println()

	if !checker.IsEnabled() {
		fmt.Println("Status: DISABLED (all commands allowed)")
		fmt.Println()
		fmt.Println("To enable restrictions, set one of these environment variables:")
		fmt.Println("  export JIRA_READONLY=1              # Read-only mode")
		fmt.Println("  export JIRA_COMMAND_ALLOWLIST=...   # Custom allowlist")
		return nil
	}

	if checker.IsReadOnly() {
		fmt.Println("Status: ENABLED (read-only mode)")
		fmt.Println("Mode:   JIRA_READONLY=1")
		fmt.Println()
		fmt.Println("Only read operations are allowed. Write commands are blocked.")
	} else {
		fmt.Println("Status: ENABLED (custom allowlist)")
		fmt.Printf("Mode:   JIRA_COMMAND_ALLOWLIST=%s\n", os.Getenv(allowlist.EnvCommandAllowlist))
		fmt.Println()
		fmt.Println("Only explicitly listed commands are allowed.")
	}

	fmt.Println()
	fmt.Println("Allowed commands:")

	allowed := checker.GetAllowedCommands()
	sort.Strings(allowed)
	for _, cmd := range allowed {
		fmt.Printf("  ✓ %s\n", cmd)
	}

	fmt.Println()
	fmt.Println("Note: 'help', 'version', '--help', '-h' are always allowed.")

	return nil
}

func runAllowlistCommands(cmd *cobra.Command, args []string) error {
	if jsonOutput {
		data := map[string]interface{}{
			"readCommands":  allowlist.ReadOnlyCommands,
			"writeCommands": allowlist.WriteCommands,
			"totalCommands": len(allowlist.ReadOnlyCommands) + len(allowlist.WriteCommands),
		}
		return outputJSON(data)
	}

	fmt.Println("Available Commands by Category")
	fmt.Println("==============================")
	fmt.Println()

	// Read commands
	fmt.Printf("READ COMMANDS (%d) - Safe for read-only mode:\n", len(allowlist.ReadOnlyCommands))
	readCmds := make([]string, len(allowlist.ReadOnlyCommands))
	copy(readCmds, allowlist.ReadOnlyCommands)
	sort.Strings(readCmds)
	for _, c := range readCmds {
		fmt.Printf("  ✓ %s\n", c)
	}

	fmt.Println()

	// Write commands
	fmt.Printf("WRITE COMMANDS (%d) - Blocked in read-only mode:\n", len(allowlist.WriteCommands))
	writeCmds := make([]string, len(allowlist.WriteCommands))
	copy(writeCmds, allowlist.WriteCommands)
	sort.Strings(writeCmds)
	for _, c := range writeCmds {
		fmt.Printf("  ✗ %s\n", c)
	}

	fmt.Println()
	fmt.Printf("Total: %d commands\n", len(allowlist.ReadOnlyCommands)+len(allowlist.WriteCommands))

	return nil
}

func runAllowlistCheck(cmd *cobra.Command, args []string) error {
	command := args[0]
	checker := allowlist.NewChecker()

	isAllowed := checker.IsAllowed(command)
	checkErr := checker.Check(command)

	if jsonOutput {
		result := map[string]interface{}{
			"command": command,
			"allowed": isAllowed,
		}
		if checkErr != nil {
			result["error"] = checkErr.Error()
		}
		return outputJSON(result)
	}

	if isAllowed {
		fmt.Printf("✓ Command '%s' is ALLOWED\n", command)
		if !checker.IsEnabled() {
			fmt.Println("  (allowlist is disabled - all commands allowed)")
		}
		return nil
	}

	fmt.Printf("✗ Command '%s' is BLOCKED\n", command)
	if checkErr != nil {
		fmt.Printf("  Reason: %s\n", checkErr.Error())
	}

	// Return exit code 1 for blocked commands (useful for scripting)
	os.Exit(1)
	return nil
}

func runAllowlistEnable(cmd *cobra.Command, args []string) {
	shell := detectShell()
	shellProfile := getShellProfile(shell)

	fmt.Println("How to Enable Command Allowlist")
	fmt.Println("===============================")
	fmt.Println()

	fmt.Println("OPTION 1: Read-Only Mode (Recommended for AI agents)")
	fmt.Println("----------------------------------------------------")
	fmt.Println("Allows only commands that read data, blocks all write operations.")
	fmt.Println()
	fmt.Println("  # Temporary (current session only):")
	fmt.Println("  export JIRA_READONLY=1")
	fmt.Println()
	fmt.Printf("  # Permanent (add to %s):\n", shellProfile)
	fmt.Println("  echo 'export JIRA_READONLY=1' >> " + shellProfile)
	fmt.Println("  source " + shellProfile)
	fmt.Println()

	fmt.Println("OPTION 2: Custom Allowlist")
	fmt.Println("--------------------------")
	fmt.Println("Allow only specific commands (comma-separated).")
	fmt.Println()
	fmt.Println("  # Temporary (current session only):")
	fmt.Println("  export JIRA_COMMAND_ALLOWLIST=\"get,search,list,fields\"")
	fmt.Println()
	fmt.Printf("  # Permanent (add to %s):\n", shellProfile)
	fmt.Println("  echo 'export JIRA_COMMAND_ALLOWLIST=\"get,search,list\"' >> " + shellProfile)
	fmt.Println("  source " + shellProfile)
	fmt.Println()

	fmt.Println("OPTION 3: Per-Command Restriction")
	fmt.Println("----------------------------------")
	fmt.Println("Apply restriction to a single command only.")
	fmt.Println()
	fmt.Println("  JIRA_READONLY=1 jcfa get ABC-123")
	fmt.Println()

	fmt.Println("TO DISABLE:")
	fmt.Println("-----------")
	fmt.Println("  unset JIRA_READONLY")
	fmt.Println("  unset JIRA_COMMAND_ALLOWLIST")
	fmt.Println()
	fmt.Printf("  # Or remove the export line from %s\n", shellProfile)
	fmt.Println()

	fmt.Println("VERIFY:")
	fmt.Println("-------")
	fmt.Println("  jcfa allowlist status    # Check current status")
	fmt.Println("  jcfa allowlist check get # Test specific command")
}

// detectShell returns the user's current shell
func detectShell() string {
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		return "zsh"
	} else if strings.Contains(shell, "bash") {
		return "bash"
	} else if strings.Contains(shell, "fish") {
		return "fish"
	}
	return "bash" // default
}

// getShellProfile returns the profile file for the given shell
func getShellProfile(shell string) string {
	home := os.Getenv("HOME")
	switch shell {
	case "zsh":
		return home + "/.zshrc"
	case "fish":
		return home + "/.config/fish/config.fish"
	default:
		return home + "/.bashrc"
	}
}
