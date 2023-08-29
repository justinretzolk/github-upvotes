package main

import (
	"github.com/shurcooL/githubv4"
)

// https://docs.github.com/en/graphql/reference/objects#organization
type Organization struct {
	Project Project `graphql:"projectV2(number: $project)"`
}

// https://docs.github.com/en/graphql/reference/objects#projectv2
type Project struct {
	Items ProjectItems `graphql:"items(first: 1, after: $projectItemsCursor)"`
}

// https://docs.github.com/en/graphql/reference/objects#projectv2itemconnection
type ProjectItems struct {
	PageInfo PageInfo      `graphql:"pageInfo"`
	Nodes    []ProjectItem `graphql:"nodes"`
}

// https://docs.github.com/en/graphql/reference/objects#pageinfo
type PageInfo struct {
	EndCursor   string `graphql:"endCursor"`
	HasNextPage bool   `graphql:"hasNextPage"`
}

const (
	ItemTypeDraftIssue  = "DRAFT_ISSUE"
	ItemTypeIssue       = "ISSUE"
	ItemTypePullRequest = "PULL_REQUEST"
)

// https://docs.github.com/en/graphql/reference/objects#projectv2item
type ProjectItem struct {
	ProjectItemId string  `graphql:"id"`
	Archived      bool    `graphql:"isArchived"`
	Content       Content `graphql:"content"`
	Type          string  `graphql:"type"`
}

// RootReactions returns the total reactions to the issue or pull request that the Project Item is connected to
func (p ProjectItem) RootReactions() int {
	if p.Type == ItemTypeIssue {
		return p.Content.Issue.Reactions.TotalCount
	}

	return p.Content.PullRequest.Reactions.TotalCount
}

// https://docs.github.com/en/graphql/reference/unions#projectv2itemcontent
type Content struct {
	Issue       IssueContentFragment       `graphql:"... on Issue"`
	PullRequest PullRequestContentFragment `graphql:"... on PullRequest"`
}

// https://docs.github.com/en/graphql/reference/objects#issue
type IssueContentFragment struct {
	Comments        ConnectionWithReactables `graphql:"comments(first: 10, after: $commentsCursor)"`
	Reactions       Reactions                `graphql:"reactions"`
	TrackedInIssues ConnectionWithReactables `graphql:"trackedInIssues(first: 10, after: $trackedInIssuesCursor)"`
	TrackedIssues   ConnectionWithReactables `graphql:"trackedIssues(first: 10, after: $trackedIssuesCursor)"`
}

// https://docs.github.com/en/graphql/reference/objects#pullrequest
type PullRequestContentFragment struct {
	Closed                  bool                     `graphql:"closed"` // Candidate for removal if it's not useful
	ClosingIssuesReferences ConnectionWithReactables `graphql:"closingIssuesReferences(first: 10, after: $closingIssuesReferencesCursor)"`
	Comments                ConnectionWithReactables `graphql:"comments(first: 10, after: $commentsCursor)"`
	Reactions               Reactions                `graphql:"reactions"`
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

// Returns true if the connection has another page
func (c ConnectionWithReactables) HasNextPage() bool {
	return c.PageInfo.HasNextPage
}

// Returns the cursor marking the end of the current page
func (c ConnectionWithReactables) EndCursor() string {
	return c.PageInfo.EndCursor
}

// Returns the total count of Reactables, as reported by the API
func (c ConnectionWithReactables) Total() int {
	return c.TotalCount
}

// Reactable is essentially an interface for objects that contain the "reactions" object
type Reactable struct {
	Reactions Reactions `graphql:"reactions"`
}

// UpvoteQuery represents the struct needed to query for the number of upvotes on a Project Item
type UpvoteQuery struct {
	Organization Organization `graphql:"organization(login: $org)"`
}

// processUpvoteQueryResponse takes a map[string]interface{} representing the variables of a query and then
// parses the response to an upvote query. The variables are updated with the appropriate cursors, and the
// function returns a boolean indicating whether there's additional pages of the connections, and two slices
// of Reactables representing linked comments or issues, respectively.
func (u *UpvoteQuery) processUpvoteQueryResponse(v map[string]interface{}) (hasNextPage bool, comments, issues []int) {

	switch item := u.Organization.Project.Items.Nodes[0]; {
	case item.Type == ItemTypeIssue:
		i := item.Content.Issue

		// add the connections to the output
		comments = append(comments, reactableUpvoteCounts(i.Comments.Nodes)...)
		issues = append(issues, reactableUpvoteCounts(i.TrackedIssues.Nodes)...)
		issues = append(issues, reactableUpvoteCounts(i.TrackedInIssues.Nodes)...)

		// if there are no more pages of any of the connections, exit early
		if i.Comments.HasNextPage() || i.TrackedIssues.HasNextPage() || i.TrackedInIssues.HasNextPage() {
			hasNextPage = true
			v["commentsCursor"] = githubv4.String(i.Comments.EndCursor())
			v["trackedInIssuesCursor"] = githubv4.String(i.TrackedInIssues.EndCursor())
			v["trackedIssuesCursor"] = githubv4.String(i.TrackedIssues.EndCursor())
		}

	case item.Type == ItemTypePullRequest:
		p := item.Content.PullRequest

		// add the connections to the output
		comments = append(comments, reactableUpvoteCounts(p.Comments.Nodes)...)
		issues = append(issues, reactableUpvoteCounts(p.ClosingIssuesReferences.Nodes)...)

		if p.Comments.HasNextPage() || p.ClosingIssuesReferences.HasNextPage() {
			hasNextPage = true
			v["commentsCursor"] = githubv4.String(p.Comments.EndCursor())
			v["closingIssuesReferencesCursor"] = githubv4.String(p.ClosingIssuesReferences.EndCursor())
		}
	}

	return
}

// reactableUpvoteCounts takes a slice of Reactables and returns a slice of ints
// representing the number of upvotes on each of the Reactables
func reactableUpvoteCounts(r []Reactable) (i []int) {
	for _, n := range r {
		i = append(i, n.Reactions.TotalCount)
	}
	return
}
