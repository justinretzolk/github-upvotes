package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/justinretzolk/github-upvotes/lib/upvotes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// flags
	cursor         string
	org            string
	field_name     string
	field_id       string
	project_id     string
	project_number int
	write          bool

	rootCmd = &cobra.Command{
		Use:   "github-upvotes",
		Short: "github-upvotes calculates upvotes for items in a GitHub Project",
		Long:  ` `, // todo
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateConfig(); err != nil {
				return err
			}

			client := upvotes.NewCalculator()
			if err := client.CalculateUpvotes(); err != nil {
				return err
			}

			return nil
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
	rootCmd.PersistentFlags().StringVar(&cursor, "cursor", "", "the cursor to begin querying the project from (env: GITHUB_CURSOR)")
	rootCmd.PersistentFlags().StringVar(&org, "org", "", "organization that owns the project (env: GITHUB_ORG)")
	rootCmd.PersistentFlags().StringVar(&field_name, "field_name", "", "the name of the project field used to track upvotes (env: GITHUB_FIELD_NAME)")
	rootCmd.PersistentFlags().StringVar(&field_id, "field_id", "", "the id of the project field used to track upvotes (env: GITHUB_FIELD_ID)")
	rootCmd.PersistentFlags().StringVar(&project_id, "project_id", "", "the id of the project to query (env: GITHUB_PROJECT_ID)")
	rootCmd.PersistentFlags().IntVar(&project_number, "project_number", 0, "the number of the project to query (env: GITHUB_PROJECT_NUMBER)")
	rootCmd.PersistentFlags().BoolVar(&write, "write", false, "update the project with the results (env: GITHUB_WRITE)")

	// viper bindings
	viper.BindPFlag("cursor", rootCmd.Flags().Lookup("cursor"))
	viper.BindPFlag("org", rootCmd.Flags().Lookup("org"))
	viper.BindPFlag("field_name", rootCmd.Flags().Lookup("field_name"))
	viper.BindPFlag("field_id", rootCmd.Flags().Lookup("field_id"))
	viper.BindPFlag("project_id", rootCmd.Flags().Lookup("project_id"))
	viper.BindPFlag("project_number", rootCmd.Flags().Lookup("project_number"))
	viper.BindPFlag("write", rootCmd.Flags().Lookup("update"))
	viper.SetEnvPrefix("GITHUB")
	viper.AutomaticEnv()
}

func validateConfig() error {
	var missing []string
	for _, v := range []string{"org", "project_number", "field_name", "token"} {
		if !viper.IsSet(v) {
			missing = append(missing, v)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf(fmt.Sprintf("missing required configuration: %v", missing))
	}

	slog.Info("configuration successfully loaded")
	return nil
}
