package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// Required variables
const (
	GitHubTokenEnvironmentVariable         = "GITHUB_TOKEN"
	GitHubOrganizationEnvironmentVariable  = "GITHUB_ORGANIZATION"
	GitHubOutputEnvironmentVariable        = "GITHUB_OUTPUT"
	GitHubProjectNumberEnvironmentVariable = "PROJECT_NUMBER"
	GitHubCursorEnvironmentVariable        = "CURSOR"
)

// EnvironmentVariables_Values returns all elements of the EnvironmentVariables enum
func EnvironmentVariables_Values() []string {
	return []string{
		GitHubTokenEnvironmentVariable,
		GitHubOrganizationEnvironmentVariable,
		GitHubOutputEnvironmentVariable,
		GitHubProjectNumberEnvironmentVariable,
	}
}

func main() {

	// Validate the environment
	if err := validateEnv(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	// Metrics calculation (runtime)
	defer timer(time.Now())()

	// Ensure that a value is always output for cursor data
	var cursor *githubv4.String
	if c, ok := os.LookupEnv(GitHubCursorEnvironmentVariable); ok {
		cursor = githubv4.NewString(githubv4.String(c))
	}
	defer output(cursor, os.Getenv(GitHubOutputEnvironmentVariable))()

	// Enable Debug Logging
	// The existence of these env vars is enough to trigger debug in Actions, so will here too
	_, runnerDebug := os.LookupEnv("RUNNER_DEBUG")
	_, stepDebug := os.LookupEnv("STEP_DEBUG")
	if runnerDebug || stepDebug {
		opts := &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}

		logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
		slog.SetDefault(logger)
	}

	// Create a connection to GitHub
	token := os.Getenv(GitHubTokenEnvironmentVariable)
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), src)
	gh := githubv4.NewClient(httpClient)

	// Instantiate a GitHubProject
	projectInt, err := strconv.Atoi(os.Getenv(GitHubProjectNumberEnvironmentVariable))
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	project, err := NewGitHubProject(gh, os.Getenv(GitHubOrganizationEnvironmentVariable), projectInt)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	if err := project.UpdateUpvotes(gh, cursor); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	/*
		ctx := context.Background()
		c, err := NewCalculator(ctx)
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}

		err = c.UpdateUpvotes(ctx)
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
	*/
}

// validateEnv validates that the required variables are set
func validateEnv() error {
	var missing []string

	for _, k := range EnvironmentVariables_Values() {

		if _, ok := os.LookupEnv(k); !ok {
			missing = append(missing, k)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missing)
	}

	return nil
}

// timer helps measure the time that the program takes to run
func timer(t time.Time) func() {
	return func() {
		slog.Info(fmt.Sprintf("elapsed time: %v", time.Since(t)))
	}
}

// output writes the cursor to the file at $GITHUB_OUTPUT
func output(cursor *githubv4.String, path string) func() {
	return func() {
		var c string
		if cursor != nil {
			c = (string)(*cursor)
		}

		path := os.Getenv("GITHUB_OUTPUT")
		output := fmt.Sprintf("cursor=%s\n", c)

		outputFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}

		defer outputFile.Close()
		if _, err := outputFile.WriteString(output); err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
	}
}
