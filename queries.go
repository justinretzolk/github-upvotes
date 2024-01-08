package main

import "github.com/shurcooL/githubv4"

// ProjectItemsQuery is used to list the project items in a project
type ProjectItemsQuery struct {
	ProjectV2ObjectFragment `graphql:"node(id: $nodeId)"`
}

// HasNextPage returns true if there are additional project items to be listed
func (p ProjectItemsQuery) HasNextPage() bool {
	return p.Items.HasNextPage
}

// ProjectV2ObjectFragment is an intermediary fragment used for selecting the ProjectV2 object
type ProjectV2ObjectFragment struct {
	ProjectFragment `graphql:"...on ProjectV2"`
}

// ProjectFragment represents a ProjectV2 object
type ProjectFragment struct {
	Items ProjectItemsFragment `graphql:"items(first:10, after: $cursor)"`
}

// ProjectItemsFragment is used as an embedded struct in ProjectFragment, and represents
// the information about the items in a project
type ProjectItemsFragment struct {
	PageInfo `graphql:"pageInfo"`
	Edges    []ProjectItemEdgeFragment
}

// PageInfo represents pagingation information returned by GitHub's GraphQL API
type PageInfo struct {
	EndCursor   githubv4.String
	HasNextPage bool
}

// ProjectItemEdgeFragment represents the connection between a project and a project item
type ProjectItemEdgeFragment struct {
	Cursor              githubv4.String
	ProjectItemFragment `graphql:"node"`
}

// ProjectItemFragment represents a node at the end of a ProjectItemEdge
type ProjectItemFragment struct {
	Id           githubv4.ID
	IsArchived   bool
	Type         string
	UpvotesField struct {
		ProjectV2ItemFieldNumberValueFragment `graphql:"...on ProjectV2ItemFieldNumberValue"`
	} `graphql:"upvotesField: fieldValueByName(name:\"Upvotes\")"` // todo: reconsider opinionated field name
	UpvotesCursorField struct {
		ProjectV2ItemFieldTextValueFragment `graphql:"...on ProjectV2ItemFieldTextValue"`
	} `graphql:"upvotesCursorField: fieldValueByName(name:\"Upvotes_Cursor\")"` // todo: reconsider opinionated field name
	Content Content
}

// GetContent returns the issue or pull request that is connected to the project item
func (p ProjectItemFragment) GetContent() ContentFragment {
	var content ContentFragment

	switch p.Content.Type {
	case "Issue":
		content = p.Content.Issue
	case "PullRequest":
		content = p.Content.PullRequest
	}

	return content
}

// Skip returns true if upvotes should not be calculated for the project item. A project item should
// be skipped if it meets any of these criterea:
//
// - It is a draft item
// - The item is archived
// - The issue or pull request connected to the project item is closed
// - There are no new timeline items since the existing cursor
func (p ProjectItemFragment) Skip() bool {
	return p.Type == "DraftIssue" || p.IsArchived || p.GetContent().Closed || !p.current()
}

// Current returns true if there have been no updates since the last run
func (p ProjectItemFragment) current() bool {
	return p.UpvotesCursorField.Value != p.GetContent().TimelineItems.EndCursor
}

// ProjectV2ItemFieldNumberValueFragment is used to get the value of a number field in a project
type ProjectV2ItemFieldNumberValueFragment struct {
	Value float64 `graphql:"number"`
}

// ProjectV2ItemFieldTextValueFragment is used to get the value of a text field in a project
type ProjectV2ItemFieldTextValueFragment struct {
	Value githubv4.String `graphql:"text"`
}

// Content is the actual Issue or Pull Request connected to a Project Item
type Content struct {
	Type        string          `graphql:"__typename"`
	Issue       ContentFragment `graphql:"...on Issue"`
	PullRequest ContentFragment `graphql:"...on PullRequest"`
}

// Common content fragment represents an Issue or Pull Request.
type ContentFragment struct {
	CommentsAndReactionsFragment
	Id     githubv4.String
	Closed bool

	TimelineItems struct {
		PageInfo `graphql:"pageInfo"`
		Nodes    []TimelineItem
	} `graphql:"timelineItems(first: 10, after: $timelineCursor, itemTypes: [CONNECTED_EVENT, CROSS_REFERENCED_EVENT, ISSUE_COMMENT, MARKED_AS_DUPLICATE_EVENT, REFERENCED_EVENT, SUBSCRIBED_EVENT])"`
}

// Upvotes returns the total upvotes for the Issue or Pull Request
func (c ContentFragment) Upvotes() int {
	upvotes := c.countOfCommentsAndReactions()

	for _, node := range c.TimelineItems.Nodes {
		upvotes += node.upvotes()
	}

	return upvotes
}

// CommentsAndReactionsFragment is embedded to add the Comments and Reactions fields
type CommentsAndReactionsFragment struct {
	Comments  TotalCountFragment
	Reactions TotalCountFragment
}

// countOfCommentsAndReactions returns the number of comments on and reactions to the item.
func (c CommentsAndReactionsFragment) countOfCommentsAndReactions() int {
	return c.Comments.TotalCount + c.Reactions.TotalCount
}

// TotalCountFragment is used as a general purpose fragment when the only needed information is
// the total count of connections.
type TotalCountFragment struct {
	TotalCount int
}

// TimelineItem respresents an individual timeline item -- an event in the Issue or Pull
// Request's history.
type TimelineItem struct {
	Type                   githubv4.String                 `graphql:"__typename"`
	ConnectedEvent         ConnectedOrCrossReferencedEvent `graphql:"...on ConnectedEvent"`
	CrossReferencedEvent   ConnectedOrCrossReferencedEvent `graphql:"...on CrossReferencedEvent"`
	IssueComment           IssueComment                    `graphql:"...on IssueComment"`
	MarkedAsDuplicateEvent MarkedAsDuplicateEvent          `graphql:"...on MarkedAsDuplicateEvent"`
}

// Upvotes returns the total upvotes for the given timeline item
func (t TimelineItem) upvotes() int {
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

// IssueOrPullRequestCommentsAndReactionsFragment is embedded in the common case of separate Issue and Pull Request
// fields that are both of type CommentsAndReactionsFragment.
type IssueOrPullRequestCommentsAndReactionsFragment struct {
	Type        string                       `graphql:"__typename"`
	Issue       CommentsAndReactionsFragment `graphql:"...on Issue"`
	PullRequest CommentsAndReactionsFragment `graphql:"...on PullRequest"`
}

// upvotes returns the count of comments and reactions to the Issue or Pull Request connected to a TimelineItem
func (i IssueOrPullRequestCommentsAndReactionsFragment) upvotes() int {
	switch i.Type {
	case "Issue":
		return i.Issue.countOfCommentsAndReactions()
	case "PullRequest":
		return i.PullRequest.countOfCommentsAndReactions()
	default:
		return 0 // todo: there's probably a better way to do this
	}
}

// Represents events when an issue or pull request was connected to, or cross-referenced
// the item.
type ConnectedOrCrossReferencedEvent struct {
	IssueOrPullRequestCommentsAndReactionsFragment `graphql:"source"`
}

// Represents an event of someone commenting on the item
type IssueComment struct {
	Reactions TotalCountFragment
}

// Represents the item being marked as a duplicate of the canonical item
type MarkedAsDuplicateEvent struct {
	IssueOrPullRequestCommentsAndReactionsFragment `graphql:"canonical"`
}

// AdditionalTimelineItemQuery is used to query for additional timeline items when there
// are more than the 100 that are accounted for in the initial ProjectItemsQuery
type AdditionalTimelineItemQuery struct {
	Content   `graphql:"node(id: $nodeId)"`
	RateLimit RateLimit
}

// RateLimit represents information related to the GitHub GraphQL rate limit
type RateLimit struct {
	Remaining int
	Cost      int
}

// ProjectItemQuery is used to list the timeline items for a specific project item
type ProjectItemQuery struct {
	ProjectV2ItemObjectFragment `graphql:"node(id: $nodeId)"`
}

// HasNextPage returns true if there are additional timeline items for the project item
func (p ProjectItemQuery) HasNextPage() bool {
	return p.GetContent().TimelineItems.HasNextPage
}

// ProjectV2ItemObjectFragment is an intermediary fragment used for selecting the ProjectV2Item object
type ProjectV2ItemObjectFragment struct {
	ProjectItemFragment `graphql:"...on ProjectV2Item"`
}
