package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/shurcooL/githubv4"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

func main() {

	// always output the cursor
	defer output()

	// load the environment
	viper.SetEnvPrefix("GITHUB")
	viper.AutomaticEnv()

	if !viper.IsSet("TOKEN") {
		slog.Error("Missing required environment variable: GITHUB_TOKEN")
		os.Exit(1)
	}

	// Debug Logging
	if _, ok := os.LookupEnv("RUNNER_DEBUG"); ok {
		opts := &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}

		logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
		slog.SetDefault(logger)
	}

	// Create a connection to GitHub
	ctx := context.Background()
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: viper.GetString("TOKEN")})
	gh := githubv4.NewClient(oauth2.NewClient(ctx, src))

	project, field, err := GetProjectInfo(gh)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	if err := UpdateUpvotes(gh, ctx, project, field); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

// output writes the cursor to the file at $GITHUB_OUTPUT
func output() func() {
	return func() {
		outputFile, err := os.OpenFile(viper.GetString("OUTPUT"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}

		defer outputFile.Close()
		if _, err := outputFile.WriteString(viper.GetString("CURSOR")); err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
	}
}
