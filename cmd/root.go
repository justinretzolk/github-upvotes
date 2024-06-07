package cmd

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"

	"github.com/justinretzolk/github-upvotes/internal/project"
)

// NewGitHubClient sets up the client that will be used to communicate with GitHub
func NewGitHubClient(ctx context.Context) (*githubv4.Client, error) {
	var client *githubv4.Client

	if !viper.IsSet("token") {
		return client, errors.New("authentication token not provided")
	}

	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: viper.GetString("token")})
	client = githubv4.NewClient(oauth2.NewClient(ctx, src))

	return client, nil
}

// UpdateUpvotes is the main processing function that updates the upvote count in the Project
func UpdateUpvotes(cmd *cobra.Command) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error)

	if !viper.IsSet("project") {
		return errors.New("project id not provided")
	}

	if !viper.IsSet("field") {
		return errors.New("project field id not provided")
	}

	gh, err := NewGitHubClient(ctx)
	if err != nil {
		slog.Error("unable to generate github client", "message", err)
	}

	itemChan, wg := project.GetProjectItems(ctx, gh, viper.GetString("project"), errChan)
	updateChan := project.ProcessProjectItems(ctx, gh, itemChan, errChan)
	done := project.UpdateProjectItems(ctx, gh, wg, viper.GetString("project"), viper.GetString("field"), updateChan, errChan)

	select {
	case err := <-errChan:
		return err
	case <-done:
		break
	}

	return nil
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "github-upvotes",
	Short: "github-upvotes - calculate upvotes for GitHub Project items",
	Run: func(cmd *cobra.Command, args []string) {
		err := UpdateUpvotes(cmd)
		if err != nil {
			slog.Error(err.Error())
			return
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("field", "f", "", "ID of the custom field representing upvotes in the GitHub Project")
	rootCmd.PersistentFlags().StringP("project", "p", "", "ID of the GitHub Project")
	rootCmd.Flags().StringP("token", "t", "", "Token used to authenticate with the GitHub API")
	rootCmd.Flags().BoolP("verbose", "v", false, "Verbose output, including debug logging")

	viper.BindPFlags(rootCmd.Flags())
	viper.BindPFlags(rootCmd.PersistentFlags())

	viper.BindEnv("field", "FIELD_ID")
	viper.BindEnv("project", "PROJECT_ID")
	viper.BindEnv("token", "GITHUB_TOKEN")
	viper.BindEnv("verbose", "RUNNER_DEBUG")
}
