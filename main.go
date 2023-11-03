package main

import (
	"fmt"
	"log/slog"
	"os"
)

// init validates that the required variables are set before running
func init() {
	requiredVars := []string{"GITHUB_TOKEN", "GITHUB_ORGANIZATION", "GITHUB_OUTPUT", "PROJECT_NUMBER", "UPVOTE_FIELD_NAME"}
	var missing []string

	for _, k := range requiredVars {
		if _, ok := os.LookupEnv(k); !ok {
			missing = append(missing, k)
		}
	}

	if len(missing) > 0 {
		slog.Error(fmt.Sprintf("Missing required environment variables: %v", missing))
		os.Exit(1)
	}
}

func main() {

	c, err := NewCalculator()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	cursor, err := c.CalculateUpvotes()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	if cursor != nil {
		if err = output(*cursor); err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
	}
}

// output writes the cursor to the file at $GITHUB_OUTPUT
func output(cursor string) error {
	path := os.Getenv("GITHUB_OUTPUT")
	output := fmt.Sprintf("cursor=%s\n", cursor)

	outputFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	defer outputFile.Close()
	if _, err := outputFile.WriteString(output); err != nil {
		return err
	}

	return nil
}
