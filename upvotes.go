package main

// Upvotes returns the total upvotes for the Issue or Pull Request connected
// to a Project Item.
func Upvotes(c ContentFragment) int {
	upvotes := c.Comments.TotalCount + c.Reactions.TotalCount

	for _, node := range c.TimelineItems.Nodes {
		upvotes += TimelineItemUpvotes(node)
	}

	return upvotes
}

// Upvotes returns the total upvotes for the given timeline item
func TimelineItemUpvotes(t TimelineItem) int {
	// the fact that the timeline item exists means that the minimum upvotes is 1
	upvotes := 1

	switch t.Type {
	case "ConnectedEvent":
		upvotes += CommentsAndReactionsUpvotes(t.ConnectedEvent.GetCommentsAndReactionsFragment())
	case "CrossReferencedEvent":
		upvotes += CommentsAndReactionsUpvotes(t.CrossReferencedEvent.GetCommentsAndReactionsFragment())
	case "IssueComment":
		upvotes += t.IssueComment.Reactions.TotalCount
	case "MarkedAsDuplicateEvent":
		upvotes += CommentsAndReactionsUpvotes(t.MarkedAsDuplicateEvent.GetCommentsAndReactionsFragment())
	}

	return upvotes
}

// CommentsAndReactionsUpvotes returns the total number of comments and reactions
func CommentsAndReactionsUpvotes(c CommentsAndReactionsFragment) int {
	return c.Comments.TotalCount + c.Reactions.TotalCount
}
