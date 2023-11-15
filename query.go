package main

import "github.com/shurcooL/githubv4"

type Upvotable interface {
	Upvotes() int
}

func Upvotes(u ...Upvotable) int {
	var upvotes int

	for _, x := range u {
		upvotes += x.Upvotes()
	}

	return upvotes
}

// ProjectItemsQuery is used to retrieve a list of project items to be processed.
// A series of embedded structs are used such that it has the fields of the ProjectItems
// and RateLimit types. See the link below for additional details.
//
// https://github.com/shurcooL/githubv4#inline-fragments
type ProjectItemsQuery struct {
	Organization `graphql:"organization(login: $org)"`
	RateLimit    RateLimit
}

// RateLimit represents information related to the GitHub GraphQL rate limit
type RateLimit struct {
	Remaining int
	Cost      int
}

// Organization is a fragment to be embedded
type Organization struct {
	Project `graphql:"projectV2(number: $project)"`
}

// Project is a fragment to be embedded
type Project struct {
	ProjectItems `graphql:"items(first: 80, after: $projectItemsCursor)"`
}

// ProjectItems contains paging information and a list of individual project items to be processed
type ProjectItems struct {
	PageInfo PageInfo
	Edges    []ProjectItemEdge
}

// PageInfo represents pagingation information returned by GitHub's GraphQL API
type PageInfo struct {
	EndCursor   githubv4.String
	HasNextPage bool
}

// ProjectItemEdge represents the connection between the project and an individual project item.
// It contains cursor information, and embeds ProjectItem, representing the node at the end of the edge.
type ProjectItemEdge struct {
	Cursor      githubv4.String
	ProjectItem `graphql:"node"`
}

// ProjectItem represents an individual item in a Project
type ProjectItem struct {
	Id         githubv4.String
	IsArchived bool
	Type       string
	Field      struct {
		NumberValue `graphql:"... on ProjectV2ItemFieldNumberValue"`
	} `graphql:"fieldValueByName(name: $fieldName)"`
	Content Content
}

// Represents the value of a Number type custom field in a Project.
type NumberValue struct {
	Number float64
}

// GetContent returns the content of the Project Item
func (p ProjectItem) GetContent() ContentFragment {
	var x ContentFragment
	switch p.Type {
	case "Issue":
		x = p.Content.Issue
	default:
		x = p.Content.PullRequest
	}

	return x
}

// Skip returns true if the current project item should be skipped when calculating upvotes
func (p ProjectItem) Skip() bool {
	return p.IsArchived || p.Content.isClosed() || p.Content.Type == "DraftIssue"
}

// Content is the actual Issue or Pull Request connected to a Project Item
type Content struct {
	Type        string          `graphql:"__typename"`
	Issue       ContentFragment `graphql:"...on Issue"`
	PullRequest ContentFragment `graphql:"...on PullRequest"`
}

// Upvotes returns the result of Upvotes() for either c.Issue or c.PullRequest
// depending on the type of the content.
func (c Content) Upvotes() int {
	var content ContentFragment
	switch c.Type {
	case "Issue":
		content = c.Issue
	case "PullRequest":
		content = c.PullRequest
	}

	return content.upvotes()
}

// isClosed returns the value of the Closed field for either c.Issue or c.PullRequest
// depending on the type of the content.
func (c Content) isClosed() bool {
	var content ContentFragment
	switch c.Type {
	case "Issue":
		content = c.Issue
	case "PullRequest":
		content = c.PullRequest
	}

	return content.Closed
}

// Common content fragment represents an Issue or Pull Request.
type ContentFragment struct {
	BaseFragment
	Id     githubv4.String
	Closed bool

	TimelineItems struct {
		PageInfo PageInfo
		Nodes    []TimeLineItem
	} `graphql:"timelineItems(first: 100, itemTypes: [CONNECTED_EVENT, CROSS_REFERENCED_EVENT, ISSUE_COMMENT, MARKED_AS_DUPLICATE_EVENT, REFERENCED_EVENT, SUBSCRIBED_EVENT])"`
}

// Upvotes returns the total upvotes for the Issue or Pull Request
func (c ContentFragment) upvotes() int {
	upvotes := c.baseUpvotes()

	for _, node := range c.TimelineItems.Nodes {
		upvotes += node.upvotes()
	}

	return upvotes
}

// TimeLineItem respresents an individual timeline item -- an event in the Issue or Pull
// Request's history.
type TimeLineItem struct {
	Type                   githubv4.String                 `graphql:"__typename"`
	ConnectedEvent         ConnectedOrCrossReferencedEvent `graphql:"...on ConnectedEvent"`
	CrossReferencedEvent   ConnectedOrCrossReferencedEvent `graphql:"...on CrossReferencedEvent"`
	IssueComment           IssueComment                    `graphql:"...on IssueComment"`
	MarkedAsDuplicateEvent MarkedAsDuplicateEvent          `graphql:"...on MarkedAsDuplicateEvent"`
}

// Upvotes returns the total upvotes for the given timeline item
func (t TimeLineItem) upvotes() int {
	// the fact that the timeline item exists means that the minimum upvotes is 1
	upvotes := 1

	switch t.Type {
	case "ConnectedEvent":
		upvotes += t.ConnectedEvent.upvotes()
	case "CrossReferencedEvent":
		upvotes += t.CrossReferencedEvent.upvotes()
	case "IssueComment":
		upvotes += t.IssueComment.Reactions.TotalCount
	case "MarkedAsDuplicateEvent":
		upvotes += t.MarkedAsDuplicateEvent.upvotes()
	}

	return upvotes
}

// BaseFragment is embedded to add common fields to Issues / Pull Requests
type BaseFragment struct {
	Comments  TotalCountFragment
	Reactions TotalCountFragment
}

// TotalCountFragment is used as a general purpose fragment when the only needed information is
// the total count of connections.
type TotalCountFragment struct {
	TotalCount int
}

// BaseUpvotes returns the number of upvotes from the fields provided by the BaseFragment:
// The total number of comments on the item, and the total number of reactions to the item.
func (b BaseFragment) baseUpvotes() int {
	return b.Comments.TotalCount + b.Reactions.TotalCount
}

// CombinedBaseFragment is embedded in the common case of separate Issue and Pull Request
// fields that are both of type BaseFragment.
type CombinedBaseFragment struct {
	Type        string       `graphql:"__typename"`
	Issue       BaseFragment `graphql:"...on Issue"`
	PullRequest BaseFragment `graphql:"...on PullRequest"`
}

// upvotes returns the result of baseUpvotes() for the Issue or Pull Request in c,
// depending on c.Type.
func (c CombinedBaseFragment) upvotes() int {
	var base BaseFragment

	switch c.Type {
	case "Issue":
		base = c.Issue
	case "PullRequest":
		base = c.PullRequest
	}

	return base.baseUpvotes()
}

// Represents events when an issue or pull request was connected to, or cross-referenced
// the item.
type ConnectedOrCrossReferencedEvent struct {
	CombinedBaseFragment `graphql:"source"`
}

// Represents an event of someone commenting on the item
type IssueComment struct {
	Reactions TotalCountFragment
}

// Represents the item being marked as a duplicate of the canonical item
type MarkedAsDuplicateEvent struct {
	CombinedBaseFragment `graphql:"canonical"`
}

// TODO: Organize this better, below here is for paginating additional timeline items

// Common content fragment used for issues or pull request items
type TimelineItemQueryContentFragment struct {
	TimelineItems struct {
		PageInfo PageInfo
		Nodes    []TimeLineItem
	} `graphql:"timelineItems(first: 100, after: $cursor, itemTypes: [CONNECTED_EVENT, CROSS_REFERENCED_EVENT, ISSUE_COMMENT, MARKED_AS_DUPLICATE_EVENT, REFERENCED_EVENT, SUBSCRIBED_EVENT])"`
}
