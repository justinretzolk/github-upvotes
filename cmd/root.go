package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "github-upvotes",
	Short: "github-upvotes updates GitHub Projects to track interactions with Issues and Pull Requests",
	Long:  "TODO",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.Execute(); err != nil {
			return err
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("%s: %s\n", "rootCmd Execute error TODO", err.Error())
		os.Exit(1)
	}
}
