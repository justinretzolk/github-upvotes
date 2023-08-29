package upvotes

import (
	"context"
	"log/slog"

	"github.com/justinretzolk/github-upvotes/internal/client"
	"github.com/shurcooL/githubv4"
)

// CalculateUpvotes iterates through a Project's Project Items, calculating the number of upvotes
// for each one. It returns an error if any are encountered
func CalculateUpvotes(c *client.Client) error {

	for {
		hasNextPage, err := calculateProjectItemUpvotes(c)
		if err != nil {
			slog.Error(err.Error(), "function", "CalculateUpvotes")
		}

		if !hasNextPage {
			break
		}
	}

	return nil
}

// calculateProjectItemUpvotes uses the Client's ProjectItemCursor to get the next Project Item to check
// and returns the new ProjectItemCursor and a boolean indicating whether there are additional items to check, or an error
func calculateProjectItemUpvotes(c *client.Client) (bool, error) {
	var query UpvoteQuery

	variables := map[string]interface{}{
		"org":                           githubv4.String(c.Organization),
		"project":                       githubv4.Int(c.Project),
		"projectItemsCursor":            (*githubv4.String)(c.ProjectItemCursor),
		"commentsCursor":                (*githubv4.String)(nil),
		"trackedInIssuesCursor":         (*githubv4.String)(nil),
		"trackedIssuesCursor":           (*githubv4.String)(nil),
		"closingIssuesReferencesCursor": (*githubv4.String)(nil),
	}

	// todo: make these slices of int instead
	var (
		allComments     []int
		allLinkedIssues []int
	)

	// Run the query, paginate as needed
	for {
		err := c.Conn.Query(context.Background(), &query, variables)
		if err != nil {
			return false, err
		}

		hasNextPage, comments, issues := query.processUpvoteQueryResponse(variables)

		allComments = append(allComments, comments...)
		allLinkedIssues = append(allLinkedIssues, issues...)

		if !hasNextPage {
			break
		}
	}

	rootReactions := query.RootReactions()
	commentReactions := sum(allComments)
	linkedReactions := sum(allLinkedIssues)

	// Get whether or not there's more pages
	hasNextPage := query.Organization.Project.Items.PageInfo.HasNextPage

	// Update the ProjectItemCursor
	endCursor := query.Organization.Project.Items.PageInfo.EndCursor
	c.ProjectItemCursor = &endCursor

	slog.Info(
		"Query Successful",
		"project_item_id", query.Organization.Project.Items.Nodes[0].ProjectItemId,
		"reactions", rootReactions,
		"comment_reactions", commentReactions,
		"linked_issue_reactions", linkedReactions,
		"total_reactions", rootReactions+len(allComments)+commentReactions+linkedReactions,
		"num_comments", len(allComments),
		"num_linked", len(allLinkedIssues),
	)

	return hasNextPage, nil
}

func sum(u []int) int {
	var result int
	for _, x := range u {
		result += x
	}
	return result
}
