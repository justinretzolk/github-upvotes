package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type Calculator struct {
	client             *githubv4.Client
	org                githubv4.String
	projectNumber      githubv4.Int
	projectId          githubv4.String
	fieldName          githubv4.String
	fieldId            githubv4.String
	cursor             githubv4.String
	rateLimitRemaining int
}

// NewCalculator returns a new calculator. Required variables are loaded directly from
// the environment, as they're validated as present during init.
func NewCalculator() (*Calculator, error) {

	c := &Calculator{
		org:       githubv4.String(os.Getenv("GITHUB_ORGANIZATION")),
		fieldName: githubv4.String(os.Getenv("UPVOTE_FIELD_NAME")),
	}

	// Populate the GitHub client
	token := os.Getenv("GITHUB_TOKEN")
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), src)
	c.client = githubv4.NewClient(httpClient)

	// Populate the project number
	project, err := strconv.Atoi(os.Getenv("PROJECT_NUMBER"))
	if err != nil {
		return c, err
	}
	c.projectNumber = githubv4.Int(project)

	// Populate the project ID and field ID
	if err = c.getAdditionalProjectData(); err != nil {
		return c, err
	}

	// Optionally populate the cursor
	if cursor, ok := os.LookupEnv("CURSOR"); ok {
		c.cursor = githubv4.String(cursor)
	}

	return c, nil
}

// getAdditionalProjectData queries for the IDs necessary to make mutations to the project
// and sets their respective values on the Calculator. Also sets rate limit data early on in the program.
func (c *Calculator) getAdditionalProjectData() error {
	var query struct {
		Organization struct {
			Project struct {
				Id    githubv4.String
				Field struct {
					FieldFragment struct {
						Id githubv4.String
					} `graphql:"...on ProjectV2Field"`
				} `graphql:"field(name: $fieldName)"`
			} `graphql:"projectV2(number: $project)"`
		} `graphql:"organization(login: $org)"`
		RateLimit RateLimit
	}

	vars := map[string]interface{}{
		"org":       c.org,
		"project":   c.projectNumber,
		"fieldName": c.fieldName,
	}

	err := c.client.Query(context.Background(), &query, vars)
	if err != nil {
		return err
	}

	c.projectId = query.Organization.Project.Id
	c.fieldId = query.Organization.Project.Field.FieldFragment.Id
	c.rateLimitRemaining = query.RateLimit.Remaining
	slog.Debug("retrieved project and field ids", "project_id", c.projectId, "field_id", c.fieldId)
	return nil
}

// CalculateUpvotes iterates over the items in the project, calculating the upvotes for them.
// It returns a string representing the endCursor of the query and/or an error.
func (c *Calculator) CalculateUpvotes() error {

	defer c.calculateUpvotesMetrics()()

	var q UpvoteQuery

	vars := map[string]interface{}{
		"org":                c.org,
		"project":            c.projectNumber,
		"fieldName":          c.fieldName,
		"projectItemsCursor": &c.cursor,
	}

	for {
		if c.rateLimitRemaining < 100 {
			slog.Warn("respecting rate limit, exiting")
			break
		}

		if err := c.client.Query(context.Background(), &q, vars); err != nil {
			return err
		}

		c.rateLimitRemaining = q.RateLimit.Remaining

		pi := q.Organization.Project.ProjectItems
		slog.Debug("upvote query executed", "end_cursor", pi.PageInfo.EndCursor, "cost", q.RateLimit.Cost, "remaining", c.rateLimitRemaining)

		for _, edge := range pi.Edges {
			err := c.calculateProjectItemUpvotes(edge)
			if err != nil {
				return err
			}
		}

		if !pi.PageInfo.HasNextPage {
			slog.Info("reached end of project items list")
			c.cursor = githubv4.String("")
			break
		}
	}

	return nil
}

// calculateUpvotesMetrics helps generate metrics around rate limiting
func (c *Calculator) calculateUpvotesMetrics() func() {
	start := c.rateLimitRemaining
	return func() {
		diff := start - c.rateLimitRemaining
		slog.Info(fmt.Sprintf("total rate limit cost: %v", diff))
	}
}

// calculateProjectItemUpvotes calculates the upvotes for a given project item, including paging
// through any additional pages of TimelineItems, then makes a call to update the project item with
// the new total. It returns an error if one is received.
func (c *Calculator) calculateProjectItemUpvotes(p ProjectItemEdge) error {

	var needsRevisit bool
	defer func(b *bool) {
		update := *b
		if !update {
			c.cursor = p.Cursor
		}
	}(&needsRevisit)

	node := p.Node
	if node.Skip() {
		slog.Debug("skipping inactive project item", "item_id", node.Id, "cursor", p.Cursor)
		return nil
	}

	slog.Debug("calculating upvotes for project item id", "item_id", node.Id, "cursor", p.Cursor)

	content := node.GetContent()
	upvotes := content.BaseUpvotes()
	upvotes += content.timelineItemsUpvotes()

	if content.TimelineItems.PageInfo.HasNextPage {
		additionalUpvotes, err := c.getAdditionalTimelineItems(content.Id, content.TimelineItems.PageInfo.EndCursor)
		if err != nil {
			needsRevisit = true
			return err
		}
		upvotes += additionalUpvotes
	}

	// Update the item in the project
	if err := c.updateProjectItem(node.Id, upvotes); err != nil {
		needsRevisit = true
		return err
	}

	// If all goes well, update the cursor
	//c.cursor = p.Cursor

	return nil
}

// getAdditionalTimelineItems queries for additional timeline items on a given Issue or Pull Request.
// It takes two githubv4.Strings representing the node ID of the Issue or Pull Request, and the cursor
// for the TimelineItems page. It returns an int representing the number of upvotes calculates from the
// remaining timeline items.
func (c Calculator) getAdditionalTimelineItems(nodeId, cursor githubv4.String) (int, error) {

	var q struct {
		Node struct {
			Type        githubv4.String `graphql:"__typename"`
			Issue       ContentFragment `graphql:"...on Issue"`
			PullRequest ContentFragment `graphql:"...on PullRequest"`
		} `graphql:"node(id: $nodeId)"`
	}

	vars := map[string]interface{}{
		"nodeId": nodeId,
		"cursor": cursor,
	}

	var upvotes int

	for {
		slog.Debug("getting additional timeline items", "node_id", nodeId, "timeline_items_cursor", cursor)
		err := c.client.Query(context.Background(), &q, vars)
		if err != nil {
			return upvotes, err
		}

		var content ContentFragment
		switch q.Node.Type {
		case githubv4.String("Issue"):
			content = q.Node.Issue
		case githubv4.String("PullRequest"):
			content = q.Node.PullRequest
		}

		upvotes += content.timelineItemsUpvotes()

		if !content.TimelineItems.PageInfo.HasNextPage {
			break
		}

		vars["cursor"] = content.TimelineItems.PageInfo.EndCursor
	}

	return upvotes, nil
}

// updateProjectItem updates the upvote field value for the project item
func (c Calculator) updateProjectItem(itemId githubv4.String, upvotes int) error {

	slog.Info("updating project item upvote count", "item_id", itemId, "upvotes", upvotes)

	var mutation struct {
		UpdateProjectItemV2FieldValue struct {
			ClientMutationId string
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}

	input := githubv4.UpdateProjectV2ItemFieldValueInput{
		ProjectID: c.projectId,
		ItemID:    itemId,
		FieldID:   c.fieldId,
		Value: githubv4.ProjectV2FieldValue{
			Number: githubv4.NewFloat(githubv4.Float(upvotes)),
		},
	}

	if err := c.client.Mutate(context.Background(), &mutation, input, nil); err != nil {
		return err
	}

	return nil
}
