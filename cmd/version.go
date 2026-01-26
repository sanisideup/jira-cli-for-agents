package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version is the current version of the CLI
	Version = "0.1.0"
	// BuildDate is the date the binary was built
	BuildDate = "unknown"
	// GitCommit is the git commit hash
	GitCommit = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of jira-cli",
	Long:  `Print the version number, build date, and git commit of jira-cli.`,
	Run: func(cmd *cobra.Command, args []string) {
		if jsonOutput {
			fmt.Printf(`{"version":"%s","buildDate":"%s","gitCommit":"%s"}`+"\n", Version, BuildDate, GitCommit)
		} else {
			fmt.Printf("jira-cli version %s\n", Version)
			fmt.Printf("Build date: %s\n", BuildDate)
			fmt.Printf("Git commit: %s\n", GitCommit)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
