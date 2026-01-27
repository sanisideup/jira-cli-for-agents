package cmd

import (
	"fmt"
	"os"

	"github.com/sanisideup/jira-cli/pkg/allowlist"
	"github.com/sanisideup/jira-cli/pkg/client"
	"github.com/sanisideup/jira-cli/pkg/config"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	cfgFile    string
	jsonOutput bool
	verbose    bool
	noColor    bool

	// Global variables
	cfg              *config.Config
	jiraClient       *client.Client
	allowlistChecker *allowlist.Checker
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "jira-cli",
	Short: "A CLI tool for Jira Cloud",
	Long: `jira-cli is a command-line interface for interacting with Jira Cloud.
It provides commands for managing issues, projects, and more.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize allowlist checker
		allowlistChecker = allowlist.NewChecker()

		// Check if command is allowed (skip for help/version which are always allowed)
		if cmd.Name() != "help" && cmd.Name() != "version" {
			// Build full command path for nested commands
			cmdPath := cmd.Name()
			if cmd.Parent() != nil && cmd.Parent().Name() != "jira-cli" {
				cmdPath = cmd.Parent().Name() + " " + cmd.Name()
			}

			if err := allowlistChecker.Check(cmdPath); err != nil {
				return err
			}
		}

		// Skip config loading for commands that don't need it
		if cmd.Name() == "configure" || cmd.Name() == "version" || cmd.Name() == "help" || cmd.Name() == "template" {
			return nil
		}

		// Load configuration
		var err error
		if cfgFile != "" {
			// Load from custom config file path
			cfg, err = config.LoadFromPath(cfgFile)
		} else {
			// Load from default location (~/.jira-cli/config.yaml)
			cfg, err = config.Load()
		}

		if err != nil {
			return fmt.Errorf("failed to load config: %w\nRun 'jira-cli configure' to set up your credentials", err)
		}

		// Initialize Jira client
		jiraClient = client.New(cfg)

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
// Exit codes:
//   - 0: Success
//   - 1: Authentication failure
//   - 2: Validation error
//   - 3: API error
//   - 4: Configuration error
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		// Determine exit code based on error type
		exitCode := getExitCode(err)
		os.Exit(exitCode)
	}
}

// getExitCode determines the appropriate exit code based on the error type
func getExitCode(err error) int {
	errMsg := err.Error()

	// Check for specific error types
	if containsAny(errMsg, []string{"authentication", "auth", "credentials", "unauthorized", "401"}) {
		return 1 // Auth failure
	}

	if containsAny(errMsg, []string{"validation", "invalid", "required field", "400"}) {
		return 2 // Validation error
	}

	if containsAny(errMsg, []string{"API error", "500", "502", "503", "504"}) {
		return 3 // API error
	}

	if containsAny(errMsg, []string{"config", "configuration"}) {
		return 4 // Config error
	}

	// Default error
	return 1
}

// containsAny checks if the string contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.jira-cli/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
}
