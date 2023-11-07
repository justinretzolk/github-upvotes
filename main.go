package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/shurcooL/githubv4"
)

// init validates that the required variables are set before running
func validateEnv() error {
	requiredVars := []string{"GITHUB_TOKEN", "GITHUB_ORGANIZATION", "GITHUB_OUTPUT", "PROJECT_NUMBER", "UPVOTE_FIELD_NAME"}
	var missing []string

	for _, k := range requiredVars {
		if _, ok := os.LookupEnv(k); !ok {
			missing = append(missing, k)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missing)
	}

	return nil
}

func main() {
	defer timer(time.Now())()

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

	if err := validateEnv(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	c, err := NewCalculator()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	defer output(&c.cursor)()

	err = c.CalculateUpvotes()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func timer(t time.Time) func() {
	return func() {
		slog.Info(fmt.Sprintf("elapsed time: %v", time.Since(t)))
	}
}

// output writes the cursor to the file at $GITHUB_OUTPUT
func output(cursor *githubv4.String) func() {
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
