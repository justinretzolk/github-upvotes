package upvotes

import (
	"context"
	"log/slog"

	"github.com/shurcooL/githubv4"
	"github.com/spf13/viper"
)

var q UpvoteQuery

// CalculateUpvotes iterates through a Project's Project Items, calculating the number of upvotes
// for each one. It returns an error if any are encountered
func CalculateUpvotes(gh *githubv4.Client) error {
	for {
		hasNextPage, err := calculateProjectItemUpvotes(gh)
		if err != nil {
			slog.Error(err.Error())
		}

		if !hasNextPage {
			break
		}
	}

	return nil
}

// todo: docs rewrite
// calculateProjectItemUpvotes uses the Client's ProjectItemCursor to get the next Project Item to check
// and returns the new ProjectItemCursor and a boolean indicating whether there are additional items to check, or an error
func calculateProjectItemUpvotes(gh *githubv4.Client) (bool, error) {
	var upvotes int

	var vars = map[string]interface{}{
		"org":                           githubv4.String(viper.GetString("org")),
		"project":                       githubv4.Int(viper.GetInt("project")),
		"projectItemsCursor":            githubv4.String(viper.GetString("cursor")),
		"commentsCursor":                (*githubv4.String)(nil),
		"trackedInIssuesCursor":         (*githubv4.String)(nil),
		"trackedIssuesCursor":           (*githubv4.String)(nil),
		"closingIssuesReferencesCursor": (*githubv4.String)(nil),
	}

	for {
		err := gh.Query(context.Background(), &q, vars)
		if err != nil {
			return false, err
		}

		upvotes += q.ProjectItemConnectionsUpvotes()

		if !q.ProjectItemHasNextPage() {
			break
		}

		cursors := q.ProjectItemConnectionCursors()
		for k, v := range cursors {
			vars[k] = githubv4.String(v)
		}
	}

	upvotes += q.ProjectItemReactionsCount() + q.ProjectItemCommentCount()
	slog.Info("successfully calculated upvotes:", "project_item_id", q.GetProjectItemId(), "upvotes", upvotes)

	hasNextPage, cursor := q.HasNextPage()
	viper.Set("cursor", cursor)

	return hasNextPage, nil
}
