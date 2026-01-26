package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sanisideup/jira-cli/pkg/client"
	"github.com/sanisideup/jira-cli/pkg/config"
	"github.com/spf13/cobra"
)

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure Jira CLI credentials and settings",
	Long: `Interactive setup wizard to configure your Jira Cloud credentials.
You will need:
- Your Jira domain (e.g., yourcompany.atlassian.net)
- Your email address
- An API token (create one at https://id.atlassian.com/manage/api-tokens)`,
	RunE: runConfigure,
}

func init() {
	rootCmd.AddCommand(configureCmd)
}

func runConfigure(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Jira CLI Configuration ===")
	fmt.Println()

	// Prompt for domain
	fmt.Print("Jira domain (e.g., yourcompany.atlassian.net): ")
	domain, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read domain: %w", err)
	}
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// Prompt for email
	fmt.Print("Email address: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read email: %w", err)
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	// Prompt for API token
	fmt.Println("API token (create one at https://id.atlassian.com/manage/api-tokens):")
	fmt.Print("> ")
	apiToken, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read API token: %w", err)
	}
	apiToken = strings.TrimSpace(apiToken)
	if apiToken == "" {
		return fmt.Errorf("API token cannot be empty")
	}

	// Prompt for default project (optional)
	fmt.Print("Default project key (optional, press Enter to skip): ")
	defaultProject, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read default project: %w", err)
	}
	defaultProject = strings.TrimSpace(defaultProject)

	// Create config
	cfg := &config.Config{
		Domain:         domain,
		Email:          email,
		APIToken:       apiToken,
		DefaultProject: defaultProject,
		FieldMappings:  make(map[string]string),
	}

	// Validate credentials before saving
	fmt.Println()
	fmt.Println("Validating credentials...")
	jiraClient := client.New(cfg)
	user, err := jiraClient.ValidateCredentials()
	if err != nil {
		return fmt.Errorf("credential validation failed: %w", err)
	}

	fmt.Printf("✓ Successfully authenticated as: %s (%s)\n", user.DisplayName, user.EmailAddress)
	fmt.Println()

	// Save config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("✓ Configuration saved to: %s\n", configPath)
	fmt.Println()
	fmt.Println("You're all set! Try running 'jira-cli --help' to see available commands.")

	return nil
}
