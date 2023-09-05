package upvotes

const (
	itemTypeDraftIssue  = "DRAFT_ISSUE"
	itemTypeIssue       = "ISSUE"
	itemTypePullRequest = "PULL_REQUEST"
)

type UpvoteQuery struct {
	Organization Organization `graphql:"organization(login: $org)"`
}

func (u UpvoteQuery) GetProjectItemId() string {
	return u.Organization.Project.ProjectItems.Nodes[0].Id
}

// ProjectItemCommentCount returns the number of comments on the Issue or Pull Request connected to the current Project Item
func (u UpvoteQuery) ProjectItemCommentCount() int {
	return u.Organization.Project.ProjectItems.Nodes[0].commentsCount()
}

// ProjectItemConnectionsUpvotes returns the number of upvotes on the connections to the Issue or Pull Request connected to the current Project Item
func (u UpvoteQuery) ProjectItemConnectionsUpvotes() int {
	return u.Organization.Project.ProjectItems.Nodes[0].connectionsUpvotes()
}

// ProjectItemConnectionCursors returns a map of the cursors for each of the connection types for the Issue or Pull Request connected to the current Project Item
func (u UpvoteQuery) ProjectItemConnectionCursors() map[string]string {
	return u.Organization.Project.ProjectItems.Nodes[0].connectionsCursors()
}

// ProjectItemReactionsCount returns the number of reactions to the Issue or Pull Request connected to the current Project Item
func (u UpvoteQuery) ProjectItemReactionsCount() int {
	return u.Organization.Project.ProjectItems.Nodes[0].reactionsCount()
}

// ProjectItemHasNextPage returns true if any of the connections to the Issue or Pull Request connected to the current Project Item have additional pages of data
func (u UpvoteQuery) ProjectItemHasNextPage() bool {
	return u.Organization.Project.ProjectItems.Nodes[0].hasNextPage()
}

// HasNextPage returns true and the value of the EndCursor if there are additional Project Items to query. Otherwise it returns false and an empty string
func (u UpvoteQuery) HasNextPage() (bool, string) {
	p := u.Organization.Project.ProjectItems
	return p.hasNextPage(), p.endCursor()
}

type Organization struct {
	Project Project `graphql:"projectV2(number: $project)"`
}

type Project struct {
	Id           string
	ProjectItems ProjectItems `graphql:"items(first: 1, after: $projectItemsCursor)"`
}

type ProjectItems struct {
	PageInfo PageInfo
	Nodes    []ProjectItem
}

func (p ProjectItems) endCursor() string {
	return p.PageInfo.EndCursor
}

func (p ProjectItems) hasNextPage() bool {
	return p.PageInfo.HasNextPage
}

type PageInfo struct {
	EndCursor   string
	HasNextPage bool
}

type ProjectItem struct {
	Id         string
	IsArchived bool
	Field      Field `graphql:"fieldValueByName(name: $fieldName)"`
	Type       string
	Content    Content
}

/*
func (p ProjectItem) isClosed() bool {
	if p.Type == itemTypeIssue {
		return p.Content.Issue.Closed
	}
	return p.Content.PullRequest.Closed
}
*/

func (p ProjectItem) commentsCount() int {
	if p.Type == itemTypeIssue {
		return p.Content.Issue.commentsCount()
	}
	return p.Content.PullRequest.commentsCount()
}

func (p ProjectItem) reactionsCount() int {
	if p.Type == itemTypeIssue {
		return p.Content.Issue.reactionsCount()
	}
	return p.Content.PullRequest.reactionsCount()
}

func (p ProjectItem) connectionsUpvotes() int {
	if p.Type == itemTypeIssue {
		return p.Content.Issue.connectionsUpvotes()
	}
	return p.Content.PullRequest.connectionsUpvotes()
}

func (p ProjectItem) connectionsCursors() map[string]string {
	if p.Type == itemTypeIssue {
		return p.Content.Issue.connectionsCursors()
	}
	return p.Content.PullRequest.connectionsCursors()
}

func (p ProjectItem) hasNextPage() bool {
	if p.Type == itemTypeIssue {
		return p.Content.Issue.hasNextPage()
	}
	return p.Content.PullRequest.hasNextPage()
}

type Field struct {
	Value struct {
		Number float64 `graphql:"number"`
	} `graphql:"... on ProjectV2ItemFieldNumberValue"`
}

type Content struct {
	Issue       IssueContentFragment       `graphql:"... on Issue"`
	PullRequest PullRequestContentFragment `graphql:"... on PullRequest"`
}

type contentFragment interface {
	commentsCount() int
	reactionsCount() int
	connectionsUpvotes() int
	connectionsCursors() map[string]string
	hasNextPage() bool
}

type IssueContentFragment struct {
	CommonContentFragment
	TrackedInIssues ConnectionWithReactables `graphql:"trackedInIssues(first: 10, after: $trackedInIssuesCursor)"`
	TrackedIssues   ConnectionWithReactables `graphql:"trackedIssues(first: 10, after: $trackedIssuesCursor)"`
}

// connectionUpvotes returns the total number of reactions to all comments and linked issues
func (i IssueContentFragment) connectionsUpvotes() int {
	return i.Comments.reactableUpvotes() + i.TrackedIssues.reactableUpvotes() + i.TrackedInIssues.reactableUpvotes()
}

// connectionCursors returns the cursors for each of the connection types
func (i IssueContentFragment) connectionsCursors() map[string]string {
	return map[string]string{
		"commentsCursor":        i.Comments.endCursor(),
		"trackedInIssuesCursor": i.TrackedInIssues.endCursor(),
		"trackedIssuesCursor":   i.TrackedIssues.endCursor(),
	}
}

// hasNextPage returns true if any of the fields of IssueContentFragment have additional pages
func (i IssueContentFragment) hasNextPage() bool {
	if i.Comments.hasNextPage() || i.TrackedIssues.hasNextPage() || i.TrackedInIssues.hasNextPage() {
		return true
	}
	return false
}

type PullRequestContentFragment struct {
	CommonContentFragment
	ClosingIssuesReferences ConnectionWithReactables `graphql:"closingIssuesReferences(first: 10, after: $closingIssuesReferencesCursor)"`
}

// connectionUpvotes returns the total number of reactions to all comments and linked issues
func (p PullRequestContentFragment) connectionsUpvotes() int {
	return p.Comments.reactableUpvotes() + p.ClosingIssuesReferences.reactableUpvotes()
}

// connectionCursors returns the cursors for each of the connection types
func (p PullRequestContentFragment) connectionsCursors() map[string]string {
	return map[string]string{
		"commentsCursor":               p.Comments.endCursor(),
		"closingIssueReferencesCursor": p.ClosingIssuesReferences.endCursor(),
	}
}

// hasNextPage returns true if any of the fields of PullRequestContentFragment have additional pages
func (p PullRequestContentFragment) hasNextPage() bool {
	if p.Comments.hasNextPage() || p.ClosingIssuesReferences.hasNextPage() {
		return true
	}
	return false
}

type CommonContentFragment struct {
	Closed    bool
	Comments  ConnectionWithReactables `graphql:"comments(first: 10, after: $commentsCursor)"`
	Reactions Reactions
}

func (c CommonContentFragment) commentsCount() int {
	return c.Comments.TotalCount
}

func (c CommonContentFragment) reactionsCount() int {
	return c.Reactions.TotalCount
}

type Reactions struct {
	TotalCount int
}

// ConnectionWithReactables represents a connection to an Issue or Pull Request that
// This is basically an interface representing a connection between the item and reactable item(s).
type ConnectionWithReactables struct {
	Nodes      []Reactable
	PageInfo   PageInfo
	TotalCount int
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
	Reactions Reactions
}
