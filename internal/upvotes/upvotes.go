package upvotes

import (
	"context"
	"log/slog"

	"github.com/justinretzolk/github-upvotes/internal/client"
	"github.com/shurcooL/githubv4"
)

type CalculateUpvotesVars map[string]interface{}

// CalculateUpvotes iterates through a Project's Project Items, calculating the number of upvotes
// for each one. It returns an error if any are encountered
func CalculateUpvotes(c *client.Client) error {

	vars := CalculateUpvotesVars{
		"org":                           githubv4.String(c.Organization),
		"project":                       githubv4.Int(c.Project),
		"projectItemsCursor":            (*githubv4.String)(nil),
		"commentsCursor":                (*githubv4.String)(nil),
		"trackedInIssuesCursor":         (*githubv4.String)(nil),
		"trackedIssuesCursor":           (*githubv4.String)(nil),
		"closingIssuesReferencesCursor": (*githubv4.String)(nil),
	}

	for {
		hasNextPage, err := calculateProjectItemUpvotes(c, vars)
		if err != nil {
			slog.Error(err.Error(), "function", "CalculateUpvotes")
		}

		if !hasNextPage {
			break
		}

		vars["projectItemsCursor"] = githubv4.String(*c.ProjectItemsCursor)
	}

	return nil
}

// calculateProjectItemUpvotes uses the Client's ProjectItemCursor to get the next Project Item to check
// and returns the new ProjectItemCursor and a boolean indicating whether there are additional items to check, or an error
func calculateProjectItemUpvotes(c *client.Client, v CalculateUpvotesVars) (bool, error) {
	var (
		query   UpvoteQuery
		upvotes int
	)

	// Run the query, paginate as needed
	for {
		err := c.Conn.Query(context.Background(), &query, v)
		if err != nil {
			return false, err
		}

		reactions, hasNextPage := handleQueryPage(query, v)
		upvotes += reactions

		if !hasNextPage {
			break
		}
	}

	// reactions, number of comments to the main issue/pull request
	upvotes += query.RootTotalUpvotes() + query.RootTotalComments()

	slog.Info(
		"successful calculated upvotes:",
		"project_item_id", query.Organization.Project.Items.Nodes[0].ProjectItemId,
		"upvotes", upvotes,
	)

	// Get whether or not there's more Project Items
	hasNextPage := query.Organization.Project.Items.PageInfo.HasNextPage

	// Update the ProjectItemCursor
	c.ProjectItemsCursor = &query.Organization.Project.Items.PageInfo.EndCursor

	return hasNextPage, nil
}

// handleQueryPage handles a single page of an UpvoteQuery
func handleQueryPage(u UpvoteQuery, v CalculateUpvotesVars) (upvotes int, hasNextPage bool) {
	switch item := u.Organization.Project.Items.Nodes[0]; {
	case item.Type == itemTypeIssue:
		upvotes = item.Content.Issue.Upvotes()

		if item.Content.Issue.HasNextPage() {
			hasNextPage = true
			v["commentsCursor"] = githubv4.String(item.Content.Issue.Comments.EndCursor())
			v["trackedInIssuesCursor"] = githubv4.String(item.Content.Issue.TrackedInIssues.EndCursor())
			v["trackedIssuesCursor"] = githubv4.String(item.Content.Issue.TrackedIssues.EndCursor())
		}

	case item.Type == itemTypePullRequest:
		upvotes = item.Content.PullRequest.Upvotes()

		if item.Content.PullRequest.HasNextPage() {
			hasNextPage = true
			v["commentsCursor"] = githubv4.String(item.Content.PullRequest.Comments.EndCursor())
			v["closingIssuesReferencesCursor"] = githubv4.String(item.Content.PullRequest.ClosingIssuesReferences.EndCursor())
		}
	}

	return
}
