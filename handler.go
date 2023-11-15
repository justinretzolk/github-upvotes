package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

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

	err chan error
}

// NewCalculator returns a new calculator. Required variables are loaded directly from
// the environment, as they're validated as present during init.
func NewCalculator(ctx context.Context) (*Calculator, error) {

	c := &Calculator{
		org:       githubv4.String(os.Getenv("GITHUB_ORGANIZATION")),
		fieldName: githubv4.String(os.Getenv("UPVOTE_FIELD_NAME")),
	}

	// Populate the GitHub client
	token := os.Getenv("GITHUB_TOKEN")
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(ctx, src)
	c.client = githubv4.NewClient(httpClient)

	// Populate the project number
	project, err := strconv.Atoi(os.Getenv("PROJECT_NUMBER"))
	if err != nil {
		return c, err
	}
	c.projectNumber = githubv4.Int(project)

	// Populate the project ID and field ID
	if err = c.getAdditionalProjectData(ctx); err != nil {
		return c, err
	}

	c.err = make(chan error)

	// Optionally populate the cursor
	if cursor, ok := os.LookupEnv("CURSOR"); ok {
		c.cursor = githubv4.String(cursor)
	}

	return c, nil
}

// getAdditionalProjectData queries for the IDs necessary to make mutations to the project
// and sets their respective values on the Calculator. Also sets rate limit data early on in the program.
func (c *Calculator) getAdditionalProjectData(ctx context.Context) error {
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

	err := c.client.Query(ctx, &query, vars)
	if err != nil {
		return err
	}

	c.projectId = query.Organization.Project.Id
	c.fieldId = query.Organization.Project.Field.FieldFragment.Id
	c.rateLimitRemaining = query.RateLimit.Remaining
	slog.Debug("retrieved project and field ids", "project_id", c.projectId, "field_id", c.fieldId)
	return nil
}

// Used to track rate limiting information. Calls to this might happen out of order due to concurrency,
// so it first checks that the value passed is actually lower than the existing value.
func (c *Calculator) setRateLimitData(v int) {
	if c.rateLimitRemaining > v {
		c.rateLimitRemaining = v
	}
}

// calculateUpvotesMetrics helps generate metrics around rate limiting
func (c *Calculator) calculateUpvotesMetrics(start int) func() {
	return func() {
		diff := start - c.rateLimitRemaining
		slog.Info(fmt.Sprintf("total rate limit cost: %v", diff))
	}
}

// UpdateUpvotes iterates over the items in the project, calculating the upvotes for them, and then
// updating the project item's upvote field with the new number.
func (c *Calculator) UpdateUpvotes(ctx context.Context) error {

	// Metrics around GraphQL rate limit usage
	defer c.calculateUpvotesMetrics(c.rateLimitRemaining)()

	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start the project item update service, passing it the trigger to end the loop
	exitTrigger := make(chan struct{})
	queue := c.projectItemUpdateService(childCtx, exitTrigger)

	// Start the project item query service
	go c.projectItemQueryService(childCtx, queue)

	for {
		select {
		case <-childCtx.Done():
			return nil

		case err := <-c.err:
			return err

		case <-exitTrigger:
			return nil
		}
	}
}

func (c *Calculator) projectItemQueryService(ctx context.Context, queue chan<- updateProjectItemInput) {

	defer close(queue)

	var q ProjectItemsQuery
	vars := map[string]interface{}{
		"org":                c.org,
		"project":            c.projectNumber,
		"fieldName":          c.fieldName,
		"projectItemsCursor": &c.cursor,
	}

pageloop:
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Stop processing additional pages of project items if the reamining API credits are less than
			// twice the length of the current queue. Each update costs 1 API credit, so this should leave
			// plenty of bufffer for enqueued items to finish up. Breaking here allows the already-retrieved
			// items to process before returning
			if c.rateLimitRemaining < (2 * len(queue)) {
				slog.Debug("nearing rate limit, no further project item pages will be queried")
				break pageloop
			}

			if err := c.client.Query(ctx, &q, vars); err != nil {
				c.err <- err
				return
			}

			pi := q.Organization.Project.ProjectItems
			slog.Debug("upvote query executed", "end_cursor", pi.PageInfo.EndCursor, "cost", q.RateLimit.Cost, "remaining", c.rateLimitRemaining)

			// Update rate limit information
			c.setRateLimitData(q.RateLimit.Remaining)

			if err := c.processProjectItems(ctx, queue, pi.Edges); err != nil {
				c.err <- err
				return
			}

			if !pi.PageInfo.HasNextPage {
				c.cursor = githubv4.String("")
				return
			}
		}
	}

}

// processProjectItems takes a Context and slice of ProjectItemEdge, and processes each ProjectItemEdge
// until it's been updated. An error is returned if one is received during processing.
func (c *Calculator) processProjectItems(ctx context.Context, queue chan<- updateProjectItemInput, items []ProjectItemEdge) error {

	childCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	var wg sync.WaitGroup
	for _, item := range items {
		wg.Add(1)
		go func(e ProjectItemEdge) {
			defer wg.Done()

			select {
			case <-childCtx.Done():
				return
			default:
				if err := c.processProjectItem(childCtx, queue, e); err != nil {
					cancel(err)
				}
			}
		}(item)
	}
	wg.Wait()

	return context.Cause(childCtx)
}

// calculateProjectItemUpvotes calculates the upvotes for a given project item, including paging
// through any additional pages of TimelineItems, then sends the information to update the project item
// to the queue channel. It returns an error if one is received.
func (c *Calculator) processProjectItem(ctx context.Context, queue chan<- updateProjectItemInput, p ProjectItemEdge) error {
	var upvotes int

	if p.Skip() {
		slog.Debug("skipping inactive project item", "item_id", p.Id, "cursor", p.Cursor)

		// The cursor *should* be incremented here, but realistically, these process way faster
		// than the ones that actually need to be updated, so for now, skipped items don't update the cursor.
		// Maybe this can be revisted later.
		// c.cursor = p.Cursor
		return nil
	}

	slog.Debug("calculating upvotes for project item id", "item_id", p.Id, "cursor", p.Cursor)
	content := p.GetContent()
	upvotes += content.Upvotes()

	if content.TimelineItems.PageInfo.HasNextPage {
		slog.Debug("project item has additional timeline items", "item_id", p.Id)

		additionalUpvotes, err := c.getAdditionalTimelineItems(ctx, content.Id, content.TimelineItems.PageInfo.EndCursor)
		if err != nil {
			return err
		}
		upvotes += additionalUpvotes
	}

	queue <- updateProjectItemInput{
		item:    p,
		upvotes: upvotes,
	}

	return nil
}

// getAdditionalTimelineItems queries for additional timeline items on a given Issue or Pull Request.
// It takes two githubv4.Strings representing the node ID of the Issue or Pull Request, and the cursor
// for the TimelineItems page. It returns an int representing the number of upvotes calculates from the
// remaining timeline items.
func (c *Calculator) getAdditionalTimelineItems(ctx context.Context, nodeId, cursor githubv4.String) (int, error) {

	var upvotes int

	var q struct {
		Node struct {
			Type        githubv4.String `graphql:"__typename"`
			Issue       ContentFragment `graphql:"...on Issue"`
			PullRequest ContentFragment `graphql:"...on PullRequest"`
		} `graphql:"node(id: $nodeId)"`
		RateLimit RateLimit
	}

	vars := map[string]interface{}{
		"nodeId": nodeId,
		"cursor": cursor,
	}

	for {
		slog.Debug("getting additional timeline items", "node_id", nodeId, "timeline_items_cursor", cursor)
		err := c.client.Query(ctx, &q, vars)
		if err != nil {
			return upvotes, err
		}

		var content ContentFragment
		if q.Node.Type == githubv4.String("Issue") {
			content = q.Node.Issue
		} else {
			content = q.Node.PullRequest
		}

		upvotes += content.Upvotes()
		c.setRateLimitData(q.RateLimit.Remaining)

		if !content.TimelineItems.PageInfo.HasNextPage {
			break
		}

		vars["cursor"] = content.TimelineItems.PageInfo.EndCursor
	}

	return upvotes, nil
}

// projectItemUpdateService receives requests and updates project items. It returns an error to the error
// channel if one is received.
func (c *Calculator) projectItemUpdateService(ctx context.Context, exitTrigger chan struct{}) chan<- updateProjectItemInput {

	queue := make(chan updateProjectItemInput)

	go func() {
		for rcvd := range queue {
			select {
			case <-ctx.Done():
				return
			default:
				slog.Info("updating project item upvote count", "item_id", rcvd.item.Id, "upvotes", rcvd.upvotes)

				var mutation struct {
					UpdateProjectItemV2FieldValue struct {
						ClientMutationId string
					} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
				}

				input := githubv4.UpdateProjectV2ItemFieldValueInput{
					ProjectID: c.projectId,
					ItemID:    rcvd.item.Id,
					FieldID:   c.fieldId,
					Value: githubv4.ProjectV2FieldValue{
						Number: githubv4.NewFloat(githubv4.Float(rcvd.upvotes)),
					},
				}

				if err := c.client.Mutate(ctx, &mutation, input, nil); err != nil {
					c.err <- err
					return
				}

				c.cursor = rcvd.item.Cursor
				c.rateLimitRemaining = c.rateLimitRemaining - 1 // mutations take 1 credit, but don't return that

				// https://docs.github.com/en/graphql/overview/rate-limits-and-node-limits-for-the-graphql-api#staying-under-the-rate-limit
				time.Sleep(200 * time.Millisecond)
			}
		}

		close(exitTrigger)
	}()

	return queue
}

// Input type for updateProjectItem
type updateProjectItemInput struct {
	item    ProjectItemEdge
	upvotes int
}
