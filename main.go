package main

// TODO
// - [x] Read GH token, Project info from env
// - [ ] Iterate through the items
// 		- [ ] Read the item, handle pagination to get totals
//		- [ ] Update the Project Item's Upvote count

// FUTURE
// - [ ] Accept flags for Project info
// - [ ] Allow notification if there's a delta above a certain limit

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/justinretzolk/github-upvotes/internal/client"
	"github.com/justinretzolk/github-upvotes/internal/upvotes"
)

func init() {
	var missing []string
	env := []string{"GITHUB_ORGANIZATION", "GITHUB_PROJECT_NUMBER", "GITHUB_TOKEN"}

	for _, v := range env {
		if _, ok := os.LookupEnv(v); !ok {
			missing = append(missing, v)
		}
	}

	if len(missing) > 0 {
		slog.Error(fmt.Sprintf("missing environment variables: %v", missing))
		os.Exit(1)
	}
}

func main() {

	client, err := client.NewClient()
	if err != nil {
		slog.Error(err.Error())
	}

	err = upvotes.CalculateUpvotes(client)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

}
