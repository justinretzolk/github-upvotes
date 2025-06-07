package cmd

import (
	"fmt"
	"os"

	"github.com/justinretzolk/github-upvotes/pkg/update"
	"github.com/spf13/cobra"
)

var githubToken string

func init() {
	rootCmd.AddCommand(updateCmd)
	cobra.OnInitialize(loadConfig)
	updateCmd.Flags().StringVar(&githubToken, "token", "", "The GitHub token to use for authentication")
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates the GitHub Project with the calculated upvotes",
	Long:  "TODO",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := update.UpdateUpvotes(cmd.Flag("token").Value.String())
		return err
	},
}

func loadConfig() {
	token, ok := os.LookupEnv("GITHUB_TOKEN")
	if !ok {
		fmt.Println("GITHUB_TOKEN environment variable not set")
		os.Exit(1)
	}
	githubToken = token
}
