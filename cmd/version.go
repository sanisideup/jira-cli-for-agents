package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version is the current version of the CLI
	Version = "1.3.0"
	// BuildDate is the date the binary was built
	BuildDate = "unknown"
	// GitCommit is the git commit hash
	GitCommit = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of jcfa",
	Long:  `Print the version number, build date, and git commit of jcfa (Jira CLI for Agents).`,
	Run: func(cmd *cobra.Command, args []string) {
		if jsonOutput {
			fmt.Printf(`{"version":"%s","buildDate":"%s","gitCommit":"%s"}`+"\n", Version, BuildDate, GitCommit)
		} else {
			fmt.Printf("jcfa version %s\n", Version)
			fmt.Printf("Build date: %s\n", BuildDate)
			fmt.Printf("Git commit: %s\n", GitCommit)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
