package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/justinretzolk/github-upvotes/lib/upvotes"
	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

var (
	// flags
	cursor  string
	org     string
	project int
	write   bool

	rootCmd = &cobra.Command{
		Use:   "github-upvotes",
		Short: "github-upvotes calculates upvotes for items in a GitHub Project",
		Long:  ` `, // todo
		Run: func(cmd *cobra.Command, args []string) {
			if err := validateConfig(); err != nil {
				slog.Error(err.Error())
				return
			}

			src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: viper.GetString("token")})
			httpClient := oauth2.NewClient(context.Background(), src)
			client := githubv4.NewClient(httpClient)

			if err := upvotes.CalculateUpvotes(client); err != nil {
				slog.Error(err.Error())
				return
			}
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func init() {
	// flags for inputs, org, project, token, cursor
	rootCmd.PersistentFlags().StringVar(&cursor, "cursor", "", "the cursor to begin querying the project from")
	rootCmd.PersistentFlags().BoolVar(&write, "write", false, "update the project with the results")
	rootCmd.PersistentFlags().StringVar(&org, "org", "", "organization that owns the project")
	rootCmd.PersistentFlags().IntVar(&project, "project", 0, "the number of the project to query")

	// viper bindings
	viper.BindPFlag("cursor", rootCmd.Flags().Lookup("cursor"))
	viper.BindPFlag("org", rootCmd.Flags().Lookup("org"))
	viper.BindPFlag("project", rootCmd.Flags().Lookup("project"))
	viper.BindPFlag("write", rootCmd.Flags().Lookup("update"))

	viper.SetEnvPrefix("GITHUB")
	viper.AutomaticEnv()
}

func validateConfig() error {
	var missing []string
	for _, v := range []string{"org", "project", "token"} {
		if !viper.IsSet(v) {
			missing = append(missing, v)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf(fmt.Sprintf("missing required configuration: %v", missing))
	}
	return nil
}
