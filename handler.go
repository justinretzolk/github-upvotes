package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type Calculator struct {
	client        *githubv4.Client
	org           githubv4.String
	projectNumber githubv4.Int
	projectId     *githubv4.String
	fieldName     githubv4.String
	fieldId       *githubv4.String
	cursor        *githubv4.String
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

	project, err := strconv.Atoi(os.Getenv("PROJECT_NUMBER"))
	if err != nil {
		return c, err
	}
	c.projectNumber = githubv4.Int(project)

	if cursor, ok := os.LookupEnv("CURSOR"); ok {
		c.cursor = githubv4.NewString(githubv4.String(cursor))
	}

	return c, nil
}

// GetUpvoteQueryVars converts the values of its structs to a structure usable in an upvote query
func (c Calculator) GetUpvoteQueryVars() map[string]interface{} {
	vars := map[string]interface{}{
		"org":                 c.org,
		"project":             c.projectNumber,
		"fieldName":           c.fieldName,
		"projectItemsCursor":  (*githubv4.String)(nil),
		"timelineItemsCursor": (*githubv4.String)(nil),
	}

	if c.cursor != nil {
		vars["projectItemsCursor"] = c.cursor
	}

	return vars
}

// GetCursor returns the end cursor of the most recent page of the query, used for pagination.
func (c Calculator) GetCursor() *string {
	return (*string)(c.cursor)
}

// CalculateUpvotes iterates over the items in the project, calculating the upvotes for them
func (c Calculator) CalculateUpvotes() (*string, error) {

	for {
		hasNextPage, rateLimited, err := c.calculateProjectItemUpvotes()
		if err != nil {
			return c.GetCursor(), err
		}

		if !hasNextPage {
			break
		}

		if rateLimited {
			slog.Info("rate limit exceeded, exiting")
			return c.GetCursor(), nil
		}
	}

	return nil, nil
}

func (c *Calculator) calculateProjectItemUpvotes() (hasNextPage, rateLimited bool, err error) {

	var (
		q       UpvoteQuery
		upvotes int
		skipped bool
	)

	vars := c.GetUpvoteQueryVars()

	// Loop over the item to gather all timeline items
	for {
		err = c.client.Query(context.Background(), &q, vars)
		if err != nil {
			break // break rather than return, in order to gather pagination and rate limit information
		}

		if q.RateLimit.Remaining < 10 {
			rateLimited = true
			break
		}

		if q.Skip() {
			slog.Info("skipping inactive project item", "item_id", q.GetProjectItemId())
			skipped = true
			break
		}

		content := q.GetContent()
		upvotes += content.Upvotes()

		if !content.TimelineItems.PageInfo.HasNextPage {
			upvotes += content.BaseUpvotes() // Upvotes on the Issue or Pull Request itself
			break
		}

		slog.Info("additional timeline items exist, continuing...")
		vars["timelineItemsCursor"] = content.TimelineItems.PageInfo.EndCursor
	}

	if hasNextPage = q.Organization.Project.ProjectItems.PageInfo.HasNextPage; hasNextPage {
		c.cursor = githubv4.NewString(q.Organization.Project.ProjectItems.PageInfo.EndCursor)
	}

	if rateLimited || skipped || err != nil {
		return
	}

	// Update the item in the project
	c.setMutationData(q.Organization.Project.Id, q.Organization.Project.Field.Id)
	itemId := q.Organization.Project.ProjectItems.Nodes[0].Id
	if err = c.updateProjectItem(itemId, upvotes); err != nil {
		return
	}

	slog.Info("calculateProjectItemUpvotes", "rl_remaining", q.RateLimit.Remaining, "item_id", itemId, "endCursor", *c.GetCursor(), "upvotes", upvotes)

	return
}

// setMutationData sets the fields necessary to make a mutation call or returns
// early if the information is already set. TODO: This is going to be more expensive
// than necessary, and should be refactored to be more efficient at a later time (e.g.
// a separate function that's not called in a loop lol)
func (c *Calculator) setMutationData(projectId, fieldId githubv4.String) {
	if c.projectId == nil {
		c.projectId = &projectId
	}
	if c.fieldId == nil {
		c.fieldId = &fieldId
	}
}

// updateProjectItem updates the upvote field value for the project item
func (c Calculator) updateProjectItem(itemId githubv4.String, upvotes int) error {

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
