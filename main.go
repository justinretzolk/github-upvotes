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

	if err := validateEnv(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	// setup github client
	ctx := context.Background()
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: viper.GetString("TOKEN")})
	gh := githubv4.NewClient(oauth2.NewClient(ctx, src))

	// context for early exit
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// channel for capturing errors
	errChan := make(chan error)

	// load project data
	project := githubv4.ID(viper.GetString("PROJECT_ID"))
	field := githubv4.ID(viper.GetString("FIELD_ID"))

	// start the pipeline
	itemChan, wg := GetProjectItems(childCtx, gh, project, errChan)
	updateChan := ProcessProjectItems(childCtx, gh, itemChan, errChan)
	done := UpdateProjectItems(childCtx, gh, wg, project, field, updateChan, errChan)

	select {
	case err := <-errChan:
		cancel()
		slog.Error(err.Error())
	case <-done:
		break
	}
}
