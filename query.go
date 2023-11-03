package main

import "github.com/shurcooL/githubv4"

type UpvoteQuery struct {
	Organization struct {
		Project struct {
			Id    githubv4.String
			Field struct {
				FieldFragment `graphql:"...on ProjectV2Field"`
			} `graphql:"field(name: $fieldName)"`
			ProjectItems ProjectItems `graphql:"items(first: 1, after: $projectItemsCursor)"`
		} `graphql:"projectV2(number: $project)"`
	} `graphql:"organization(login: $org)"`
	RateLimit struct {
		Remaining int `graphql:"remaining"`
		Cost      int `graphql:"cost"`
	} `graphql:"rateLimit"`
}

type FieldFragment struct {
	Id githubv4.String
}

// Skip returns true if the current project item should be skipped when calculating upvotes
func (u UpvoteQuery) Skip() bool {
	pi := u.Organization.Project.ProjectItems.Nodes[0]

	return pi.IsArchived || pi.Content.isClosed() || pi.Content.Type == "DraftIssue"
}

// GetProjectItemId returns the ID current project item
func (u UpvoteQuery) GetProjectItemId() githubv4.String {
	return u.Organization.Project.ProjectItems.Nodes[0].Id
}

// GetContent returns the ContentFragment for the Issue or Pull Request
func (u UpvoteQuery) GetContent() ContentFragment {
	c := u.Organization.Project.ProjectItems.Nodes[0].Content
	if c.Type == "Issue" {
		return c.Issue
	}
	return c.PullRequest
}

// Fragment for pagination information
type PageInfo struct {
	EndCursor   githubv4.String `graphql:"endCursor"`
	HasNextPage bool            `graphql:"hasNextPage"`
}

// ProjectItems represents information about the project items in a project
type ProjectItems struct {
	PageInfo PageInfo
	Nodes    []ProjectItem
}

// ProjectItem represents an individual item in a Project
type ProjectItem struct {
	Id         githubv4.String
	IsArchived bool
	Type       string
	Field      struct {
		Value struct {
			Id     githubv4.String
			Number float64 `graphql:"number"`
		} `graphql:"... on ProjectV2ItemFieldNumberValue"`
	} `graphql:"fieldValueByName(name: $fieldName)"`
	Content Content `graphql:"content"`
}

// Content is the actual Issue or Pull Request connected to a Project Item
type Content struct {
	Type        string          `graphql:"__typename"`
	Issue       ContentFragment `graphql:"...on Issue"`
	PullRequest ContentFragment `graphql:"...on PullRequest"`
}

// isClosed returns true if the Issue or Pull Request is closed
func (c Content) isClosed() bool {
	if c.Type == "Issue" {
		return c.Issue.Closed
	}
	return c.PullRequest.Closed
}

// Upvotes returns the total upvotes for the Issue or Pull Request tied to the Project Item
func (c Content) Upvotes() int {
	if c.Type == "Issue" {
		return c.Issue.Upvotes()
	}
	return c.PullRequest.Upvotes()
}

// GetContent returns the ContentFragment for the Issue or Pull Request
func (c Content) GetContent() ContentFragment {
	if c.Type == "Issue" {
		return c.Issue
	}
	return c.PullRequest
}

// Common content fragment used for issues or pull request items
type ContentFragment struct {
	Closed bool `graphql:"closed"`
	BaseFragment

	TimelineItems struct {
		PageInfo PageInfo
		Nodes    []TimeLineItem
	} `graphql:"timelineItems(first: 100, after: $timelineItemsCursor, itemTypes: [CONNECTED_EVENT, CROSS_REFERENCED_EVENT, ISSUE_COMMENT, MARKED_AS_DUPLICATE_EVENT, REFERENCED_EVENT, SUBSCRIBED_EVENT])"`
}

// Upvotes returns the total upvotes for the Issue or Pull Request
func (c ContentFragment) Upvotes() int {
	var upvotes int
	for _, t := range c.TimelineItems.Nodes {
		upvotes += t.Upvotes()
	}
	return upvotes
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
		upvotes = upvotes + t.ConnectedEvent.CombinedBaseUpvotes()
	case "CrossReferencedEvent":
		upvotes = upvotes + t.CrossReferencedEvent.CombinedBaseUpvotes()
	case "IssueComment":
		upvotes = upvotes + t.IssueComment.Reactions.TotalCount
	case "MarkedAsDuplicateEvent":
		upvotes = upvotes + t.MarkedAsDuplicateEvent.CombinedBaseUpvotes()
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
	Number    int                `graphql:"number"` // TODO: Revisit utility of this
	Comments  TotalCountFragment `graphql:"comments"`
	Reactions TotalCountFragment `graphql:"reactions"`
}

// BaseUpvotes returns the number of upvotes from the fields provided by the BaseFragment:
// The total number of comments on the item, and the total number of reactions to the item.
func (b BaseFragment) BaseUpvotes() int {
	return b.Comments.TotalCount + b.Reactions.TotalCount
}

// CombinedBaseFragment combines both IssueBaseFragment and PullRequestBaseFragment
// for ease of reference, since these generally only occur together
type CombinedBaseFragment struct {
	Type        string       `graphql:"__typename"`
	Issue       BaseFragment `graphql:"...on Issue"`
	PullRequest BaseFragment `graphql:"...on PullRequest"`
}

// CombinedBaseUpvotes returns the upvotes for a CombinedBaseFragment
func (c CombinedBaseFragment) CombinedBaseUpvotes() int {
	if c.Type == "Issue" {
		return c.Issue.BaseUpvotes()
	}
	return c.PullRequest.BaseUpvotes()
}

// Represents events when an issue or pull request was connected to the item
type ConnectedEvent struct {
	IsCrossRepository    bool `graphql:"isCrossRepository"` // TODO: Revisit utility of this
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
	AuthorAssociation string             `graphql:"authorAssociation"` // TODO: Revisit utility of this
	Reactions         TotalCountFragment `graphql:"reactions"`
}

// Represents the item being marked as a duplicate of the canonical item
type MarkedAsDuplicateEvent struct {
	IsCrossRepository    bool `graphql:"isCrossRepository"` // TODO: Revisit utility of this
	CombinedBaseFragment `graphql:"canonical"`
}
