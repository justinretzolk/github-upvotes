package upvotes

const (
	itemTypeDraftIssue  = "DRAFT_ISSUE"
	itemTypeIssue       = "ISSUE"
	itemTypePullRequest = "PULL_REQUEST"
)

// UpvoteQuery represents the struct needed to query for the number of upvotes on a Project Item
type UpvoteQuery struct {
	Organization struct {
		Project struct {
			Items struct {
				PageInfo PageInfo `graphql:"pageInfo"`
				Nodes    []struct {
					ProjectItemId string `graphql:"id"`
					Archived      bool   `graphql:"isArchived"`
					Type          string `graphql:"type"`
					Content       struct {
						Issue       IssueContentFragment       `graphql:"... on Issue"`
						PullRequest PullRequestContentFragment `graphql:"... on PullRequest"`
					} `graphql:"content"`
				} `graphql:"nodes"`
			} `graphql:"items(first: 1, after: $projectItemsCursor)"`
		} `graphql:"projectV2(number: $project)"`
	} `graphql:"organization(login: $org)"`
}

// RootTotalReactions returns the total reactions to the issue or pull request that the Project Item is connected to
func (u UpvoteQuery) RootTotalUpvotes() int {
	node := u.Organization.Project.Items.Nodes[0]
	if node.Type == itemTypeIssue {
		return node.Content.Issue.Reactions.TotalCount
	}
	return node.Content.PullRequest.Reactions.TotalCount
}

// RootTotalComments returns the total number of comments on a linked issue or pull request
func (u UpvoteQuery) RootTotalComments() int {
	node := u.Organization.Project.Items.Nodes[0]
	if node.Type == itemTypeIssue {
		return node.Content.Issue.Comments.TotalCount
	}
	return node.Content.PullRequest.Comments.TotalCount
}

// https://docs.github.com/en/graphql/reference/objects#pageinfo
type PageInfo struct {
	EndCursor   string `graphql:"endCursor"`
	HasNextPage bool   `graphql:"hasNextPage"`
}

// https://docs.github.com/en/graphql/reference/objects#issue
type IssueContentFragment struct {
	Comments        ConnectionWithReactables `graphql:"comments(first: 10, after: $commentsCursor)"`
	Reactions       Reactions                `graphql:"reactions"`
	TrackedInIssues ConnectionWithReactables `graphql:"trackedInIssues(first: 10, after: $trackedInIssuesCursor)"`
	TrackedIssues   ConnectionWithReactables `graphql:"trackedIssues(first: 10, after: $trackedIssuesCursor)"`
}

// HasNextPage returns true if any of the fields of IssueContentFragment have additional pages
func (i IssueContentFragment) HasNextPage() bool {
	if i.Comments.HasNextPage() || i.TrackedIssues.HasNextPage() || i.TrackedInIssues.HasNextPage() {
		return true
	}
	return false
}

// Upvotes returns the total number of reactions to all comments and linked issues
func (i IssueContentFragment) Upvotes() int {
	return i.Comments.ReactableUpvotes() + i.TrackedIssues.ReactableUpvotes() + i.TrackedInIssues.ReactableUpvotes()
}

// https://docs.github.com/en/graphql/reference/objects#pullrequest
type PullRequestContentFragment struct {
	Closed                  bool                     `graphql:"closed"` // Candidate for removal if it's not useful
	ClosingIssuesReferences ConnectionWithReactables `graphql:"closingIssuesReferences(first: 10, after: $closingIssuesReferencesCursor)"`
	Comments                ConnectionWithReactables `graphql:"comments(first: 10, after: $commentsCursor)"`
	Reactions               Reactions                `graphql:"reactions"`
}

// HasNextPage returns true if any of the fields of PullRequestContentFragment have additional pages
func (p PullRequestContentFragment) HasNextPage() bool {
	if p.Comments.HasNextPage() || p.ClosingIssuesReferences.HasNextPage() {
		return true
	}
	return false
}

// Upvotes returns the total number of reactions to all comments and linked issues
func (p PullRequestContentFragment) Upvotes() int {
	return p.Comments.ReactableUpvotes() + p.ClosingIssuesReferences.ReactableUpvotes()
}

// https://docs.github.com/en/graphql/reference/objects#reactionconnection
type Reactions struct {
	TotalCount int `graphql:"totalCount"`
}

// ConnectionWithReactables is a structure that represents a connection to an issue or pull request.
// This is basically an interface representing a connection between the item and reactable item(s).
type ConnectionWithReactables struct {
	Nodes      []Reactable `graphql:"nodes"`
	PageInfo   PageInfo    `graphql:"pageInfo"`
	TotalCount int         `graphql:"totalCount"`
}

// ReactableUpvotes returns the sum of all reactions to the Reactables within a ConnectionWithReactables
func (c ConnectionWithReactables) ReactableUpvotes() int {
	var total int
	for _, n := range c.Nodes {
		total += n.Reactions.TotalCount
	}
	return total
}

// Returns the cursor marking the end of the current page
func (c ConnectionWithReactables) EndCursor() string {
	return c.PageInfo.EndCursor
}

// Returns true if the connection has another page
func (c ConnectionWithReactables) HasNextPage() bool {
	return c.PageInfo.HasNextPage
}

// Reactable is essentially an interface for objects that contain the "reactions" object
type Reactable struct {
	Reactions Reactions `graphql:"reactions"`
}
