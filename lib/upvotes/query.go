package upvotes

const (
	itemTypeDraftIssue  = "DRAFT_ISSUE"
	itemTypeIssue       = "ISSUE"
	itemTypePullRequest = "PULL_REQUEST"
)

type UpvoteQuery struct {
	Organization Organization `graphql:"organization(login: $org)"`
}

// getProjectItems returns the ProjectItems from the query
func (u UpvoteQuery) getProjectItems() ProjectItems {
	return u.Organization.Project.ProjectItems
}

// getProjectItem returns the current Project Item from the query
func (u UpvoteQuery) getProjectItem() ProjectItem {
	return u.getProjectItems().Nodes[0]
}

// getProjectItemContent returns the Issue or Pull Request that is connected to the Project Item
func (u UpvoteQuery) getProjectItemContent() contentFragment {
	node := u.getProjectItem()
	if node.Type == itemTypeIssue {
		return node.Content.Issue
	}
	return node.Content.PullRequest
}

func (u UpvoteQuery) GetProjectItemId() string {
	return u.getProjectItem().ProjectItemId
}

// ProjectItemCommentCount returns the number of comments on the Issue or Pull Request connected to the current Project Item
func (u UpvoteQuery) ProjectItemCommentCount() int {
	return u.getProjectItemContent().commentCount()
}

// ProjectItemConnectionsUpvotes returns the number of upvotes on the connections to the Issue or Pull Request connected to the current Project Item
func (u UpvoteQuery) ProjectItemConnectionsUpvotes() int {
	return u.getProjectItemContent().connectionUpvotes()
}

// ProjectItemConnectionCursors returns a map of the cursors for each of the connection types for the Issue or Pull Request connected to the current Project Item
func (u UpvoteQuery) ProjectItemConnectionCursors() map[string]string {
	return u.getProjectItemContent().connectionCursors()
}

// ProjectItemReactionsCount returns the number of reactions to the Issue or Pull Request connected to the current Project Item
func (u UpvoteQuery) ProjectItemReactionsCount() int {
	return u.getProjectItemContent().reactionsCount()
}

// ProjectItemHasNextPage returns true if any of the connections to the Issue or Pull Request connected to the current Project Item have additional pages of data
func (u UpvoteQuery) ProjectItemHasNextPage() bool {
	return u.getProjectItemContent().hasNextPage()
}

// HasNextPage returns true and the value of the EndCursor if there are additional Project Items to query. Otherwise it returns false and an empty string
func (u UpvoteQuery) HasNextPage() (bool, string) {
	p := u.getProjectItems()
	return p.hasNextPage(), p.endCursor()
}

type Organization struct {
	Project Project `graphql:"projectV2(number: $project)"`
}

type Project struct {
	ProjectItems ProjectItems `graphql:"items(first: 1, after: $projectItemsCursor)"`
}

// ProjectItems represents the list of Project Items connected to a Project
type ProjectItems struct {
	PageInfo PageInfo      `graphql:"pageInfo"`
	Nodes    []ProjectItem `graphql:"nodes"`
}

// endCursor returns the end cursor of the ProjectItems list
func (p ProjectItems) endCursor() string {
	return p.PageInfo.EndCursor
}

// hasNextPage returns true if there are additional Project Items to query
func (p ProjectItems) hasNextPage() bool {
	return p.PageInfo.HasNextPage
}

type PageInfo struct {
	EndCursor   string `graphql:"endCursor"`
	HasNextPage bool   `graphql:"hasNextPage"`
}

type ProjectItem struct {
	ProjectItemId string  `graphql:"id"`
	Archived      bool    `graphql:"isArchived"`
	Type          string  `graphql:"type"`
	Content       Content `graphql:"content"`
}

type Content struct {
	Issue       IssueContentFragment       `graphql:"... on Issue"`
	PullRequest PullRequestContentFragment `graphql:"... on PullRequest"`
}

// contentFragment is an interface to help retrieve information about the Issue or Pull Request that is connected to the Project Item
type contentFragment interface {
	commentCount() int
	connectionUpvotes() int
	connectionCursors() map[string]string
	reactionsCount() int
	hasNextPage() bool
}

// https://docs.github.com/en/graphql/reference/objects#issue
type IssueContentFragment struct {
	Comments        ConnectionWithReactables `graphql:"comments(first: 10, after: $commentsCursor)"`
	Reactions       Reactions                `graphql:"reactions"`
	TrackedInIssues ConnectionWithReactables `graphql:"trackedInIssues(first: 10, after: $trackedInIssuesCursor)"`
	TrackedIssues   ConnectionWithReactables `graphql:"trackedIssues(first: 10, after: $trackedIssuesCursor)"`
}

// commentCount returns the total number of comments
func (i IssueContentFragment) commentCount() int {
	return i.Comments.TotalCount
}

// connectionUpvotes returns the total number of reactions to all comments and linked issues
func (i IssueContentFragment) connectionUpvotes() int {
	return i.Comments.reactableUpvotes() + i.TrackedIssues.reactableUpvotes() + i.TrackedInIssues.reactableUpvotes()
}

// connectionCursors returns the cursors for each of the connection types
func (i IssueContentFragment) connectionCursors() map[string]string {
	return map[string]string{
		"commentsCursor":        i.Comments.endCursor(),
		"trackedInIssuesCursor": i.TrackedInIssues.endCursor(),
		"trackedIssuesCursor":   i.TrackedIssues.endCursor(),
	}
}

// reactionsCount returns the total number of reactions
func (i IssueContentFragment) reactionsCount() int {
	return i.Reactions.TotalCount
}

// hasNextPage returns true if any of the fields of IssueContentFragment have additional pages
func (i IssueContentFragment) hasNextPage() bool {
	if i.Comments.hasNextPage() || i.TrackedIssues.hasNextPage() || i.TrackedInIssues.hasNextPage() {
		return true
	}
	return false
}

// https://docs.github.com/en/graphql/reference/objects#pullrequest
type PullRequestContentFragment struct {
	Closed                  bool                     `graphql:"closed"` // Candidate for removal if it's not useful
	ClosingIssuesReferences ConnectionWithReactables `graphql:"closingIssuesReferences(first: 10, after: $closingIssuesReferencesCursor)"`
	Comments                ConnectionWithReactables `graphql:"comments(first: 10, after: $commentsCursor)"`
	Reactions               Reactions                `graphql:"reactions"`
}

// commentCount returns the total number of comments
func (p PullRequestContentFragment) commentCount() int {
	return p.Comments.TotalCount
}

// connectionUpvotes returns the total number of reactions to all comments and linked issues
func (p PullRequestContentFragment) connectionUpvotes() int {
	return p.Comments.reactableUpvotes() + p.ClosingIssuesReferences.reactableUpvotes()
}

// connectionCursors returns the cursors for each of the connection types
func (p PullRequestContentFragment) connectionCursors() map[string]string {
	return map[string]string{
		"commentsCursor":               p.Comments.endCursor(),
		"closingIssueReferencesCursor": p.ClosingIssuesReferences.endCursor(),
	}
}

// reactionsCount returns the total number of reactions
func (p PullRequestContentFragment) reactionsCount() int {
	return p.Reactions.TotalCount
}

// hasNextPage returns true if any of the fields of PullRequestContentFragment have additional pages
func (p PullRequestContentFragment) hasNextPage() bool {
	if p.Comments.hasNextPage() || p.ClosingIssuesReferences.hasNextPage() {
		return true
	}
	return false
}

type Reactions struct {
	TotalCount int `graphql:"totalCount"`
}

// ConnectionWithReactables represents a connection to an Issue or Pull Request that
// This is basically an interface representing a connection between the item and reactable item(s).
type ConnectionWithReactables struct {
	Nodes      []Reactable `graphql:"nodes"`
	PageInfo   PageInfo    `graphql:"pageInfo"`
	TotalCount int         `graphql:"totalCount"`
}

// reactableUpvotes returns the sum of all reactions to the Reactables within a ConnectionWithReactables
func (c ConnectionWithReactables) reactableUpvotes() (total int) {
	for _, n := range c.Nodes {
		total += n.Reactions.TotalCount
	}
	return
}

// EndCursor returns the cursor marking the end of the current page
func (c ConnectionWithReactables) endCursor() string {
	return c.PageInfo.EndCursor
}

// HasNextPage returns true if the connection has another page
func (c ConnectionWithReactables) hasNextPage() bool {
	return c.PageInfo.HasNextPage
}

// Reactable is essentially an interface for objects that contain the "reactions" object
type Reactable struct {
	Reactions Reactions `graphql:"reactions"`
}
