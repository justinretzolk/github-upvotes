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

type UpvoteQuery struct {
	Organization struct {
		Project struct {
			ProjectItems ProjectItems `graphql:"items(first: 80, after: $projectItemsCursor)"`
		} `graphql:"projectV2(number: $project)"`
	} `graphql:"organization(login: $org)"`
	RateLimit RateLimit `graphql:"rateLimit"`
}

type RateLimit struct {
	Remaining int `graphql:"remaining"`
	Cost      int `graphql:"cost"`
}

// Fragment for pagination information
type PageInfo struct {
	EndCursor   githubv4.String `graphql:"endCursor"`
	HasNextPage bool            `graphql:"hasNextPage"`
}

// ProjectItems represents information about the project items in a project
type ProjectItems struct {
	PageInfo PageInfo
	Edges    []ProjectItemEdge
}

type ProjectItemEdge struct {
	Cursor githubv4.String
	Node   ProjectItem
}

// ProjectItem represents an individual item in a Project
type ProjectItem struct {
	Id         githubv4.String
	IsArchived bool
	Type       string
	Field      struct {
		NumberValue `graphql:"... on ProjectV2ItemFieldNumberValue"`
	} `graphql:"fieldValueByName(name: $fieldName)"`
	Content Content `graphql:"content"`
}

// Represents the value of a Number type custom field in a Project. In an Upvote query,
// this is populated with the current number of upvotes, for use in diffing
type NumberValue struct {
	Number float64 `graphql:"number"`
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

func (c Content) Upvotes() int {
	var x int
	switch c.Type {
	case "Issue":
		x += c.Issue.Upvotes()
	case "PullRequest":
		x += c.PullRequest.Upvotes()
	}

	return x
}

// isClosed returns true if the Issue or Pull Request is closed
func (c Content) isClosed() bool {
	var x bool

	switch c.Type {
	case "Issue":
		x = c.Issue.Closed
	case "PullRequest":
		x = c.PullRequest.Closed
	}

	return x
}

// Common content fragment used for issues or pull request items
type ContentFragment struct {
	Id     githubv4.String
	Closed bool `graphql:"closed"`
	BaseFragment

	TimelineItems struct {
		PageInfo PageInfo
		Nodes    []TimeLineItem
	} `graphql:"timelineItems(first: 100, itemTypes: [CONNECTED_EVENT, CROSS_REFERENCED_EVENT, ISSUE_COMMENT, MARKED_AS_DUPLICATE_EVENT, REFERENCED_EVENT, SUBSCRIBED_EVENT])"`
}

// Upvotes returns the total upvotes for the Issue or Pull Request
func (c ContentFragment) Upvotes() int {
	x := []Upvotable{
		c.BaseFragment,
	}

	for _, y := range c.TimelineItems.Nodes {
		x = append(x, y)
	}

	return Upvotes(x...)
}

type TimeLineItem struct {
	Type                   githubv4.String        `graphql:"__typename"`
	ConnectedEvent         ConnectedEvent         `graphql:"...on ConnectedEvent"`
	CrossReferencedEvent   CrossReferencedEvent   `graphql:"...on CrossReferencedEvent"`
	IssueComment           IssueComment           `graphql:"...on IssueComment"`
	MarkedAsDuplicateEvent MarkedAsDuplicateEvent `graphql:"...on MarkedAsDuplicateEvent"`
}

// Upvotes returns the total upvotes for the given timeline item
func (t TimeLineItem) Upvotes() int {
	// the fact that the timeline item exists means that the minimum upvotes is 1
	upvotes := 1

	switch t.Type {
	case "ConnectedEvent":
		upvotes += t.ConnectedEvent.Upvotes()
	case "CrossReferencedEvent":
		upvotes += t.CrossReferencedEvent.Upvotes()
	case "IssueComment":
		upvotes += t.IssueComment.Reactions.TotalCount
	case "MarkedAsDuplicateEvent":
		upvotes += t.MarkedAsDuplicateEvent.Upvotes()
	}

	return upvotes
}

// TotalCountFragment is used as a general purpose fragment when the only needed information is
// the total count of connections.
type TotalCountFragment struct {
	TotalCount int `graphql:"totalCount"`
}

// BaseFragment is used to add common fields to larger parts of the query
type BaseFragment struct {
	Comments  TotalCountFragment `graphql:"comments"`
	Reactions TotalCountFragment `graphql:"reactions"`
}

// Upvotes returns the number of upvotes from the fields provided by the BaseFragment:
// The total number of comments on the item, and the total number of reactions to the item.
func (b BaseFragment) Upvotes() int {
	return b.Comments.TotalCount + b.Reactions.TotalCount
}

// CombinedBaseFragment combines both IssueBaseFragment and PullRequestBaseFragment
// for ease of reference, since these generally only occur together
type CombinedBaseFragment struct {
	Type        string       `graphql:"__typename"`
	Issue       BaseFragment `graphql:"...on Issue"`
	PullRequest BaseFragment `graphql:"...on PullRequest"`
}

// Upvotes returns the upvotes for a CombinedBaseFragment
func (c CombinedBaseFragment) Upvotes() int {
	var x int

	switch c.Type {
	case "Issue":
		x = c.Issue.Upvotes()
	case "PullRequest":
		x = c.PullRequest.Upvotes()
	}

	return x
}

// Represents events when an issue or pull request was connected to the item
type ConnectedEvent struct {
	CombinedBaseFragment `graphql:"source"`
}

// Represents the item being cross referenced by another issue or pull request. Has almost identical
// fields to ConnectedEvent, with the addition of an indication of whether the target will be closed
// by the source.
type CrossReferencedEvent struct {
	ConnectedEvent
	WillCloseTarget bool `graphql:"willCloseTarget"`
}

// Represents an event of someone commenting on the item
type IssueComment struct {
	Reactions TotalCountFragment `graphql:"reactions"`
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
