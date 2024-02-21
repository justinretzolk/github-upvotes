package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/shurcooL/githubv4"
	"github.com/spf13/viper"
)

func GetProjectInfo(gh *githubv4.Client) (projectId githubv4.ID, fieldId githubv4.String, err error) {

	if !viper.IsSet("ORGANIZATION") {
		err = errors.New("Missing required environment variable: GITHUB_ORGANIZATION")
		return
	}

	if !viper.IsSet("PROJECT_NUMBER") {
		err = errors.New("Missing required environment variable: GITHUB_PROJECT_NUMBER")
		return
	}

	// query used to retrieve project information
	var query struct {
		Organization struct {
			Project struct {
				Id           githubv4.ID
				UpvotesField struct {
					Field struct {
						Id githubv4.String
					} `graphql:"...on ProjectV2Field"`
				} `graphql:"field(name:\"Upvotes\")"` // todo: reconsider opinionated field name
			} `graphql:"projectV2(number: $project)"`
		} `graphql:"organization(login: $organization)"`
	}

	vars := map[string]interface{}{
		"organization": githubv4.String(viper.GetString("ORGANIZATION")),
		"project":      githubv4.Int(viper.GetInt("PROJECT_NUMBER")),
	}

	if err = gh.Query(context.TODO(), &query, vars); err != nil {
		return
	}

	projectId = query.Organization.Project.Id
	fieldId = query.Organization.Project.UpvotesField.Field.Id
	slog.Info("retrieved project info", "project_id", projectId, "field_id", fieldId)
	return
}

// UpdateUpvotes is the main entrypoint for updating the upvote counts on the Project
func UpdateUpvotes(gh *githubv4.Client, ctx context.Context, projectId githubv4.ID, fieldId githubv4.String) error {

	// for pages of the list of project items
	for {
		items, err := GetProjectItems(gh, ctx, projectId)
		if err != nil {
			return err
		}

		updates, err := ProcessProjectItems(gh, ctx, items.Edges)
		if err != nil {
			return err
		}

		// Call for the updates to be updated here
		if err := UpdateProjectItems(gh, ctx, projectId, fieldId, updates); err != nil {
			return err
		}

		if !items.HasNextPage {
			viper.Set("CURSOR", "")
			break
		}
	}

	return nil
}

// GetProjectItems gets a list of Project Items that should be processed
func GetProjectItems(gh *githubv4.Client, ctx context.Context, projectId githubv4.ID) (items ProjectItemsFragment, err error) {

	var query ProjectItemsQuery
	var cursor *githubv4.String

	if c := viper.GetString("CURSOR"); c != "" {
		slog.Info("cursor loaded", "cursor", c)
		cursor = githubv4.NewString(githubv4.String(c))
	}

	variables := map[string]interface{}{
		"nodeId": projectId,
		"cursor": cursor,
		// not used here, but a required variable nonetheless
		"timelineCursor": (*githubv4.String)(nil),
	}

	slog.Info("querying for project items")
	if err = gh.Query(ctx, &query, variables); err != nil {
		return
	}

	items = query.Items

	return
}

// ProcessProjectItems iterates through a list of Project Items, calling ProcessProjectItem
// for each one. It is responsible for updating the cursor after successfully processing a Project Item.
func ProcessProjectItems(gh *githubv4.Client, ctx context.Context, items []ProjectItemEdgeFragment) ([]Update, error) {

	var updates []Update

	for _, item := range items {
		if item.Skip() {
			slog.Info("item skipped", "item_id", item.Id)
			continue
		}

		update, err := ProcessProjectItem(gh, ctx, item)
		if err != nil {
			return updates, err
		}

		updates = append(updates, update)
	}

	return updates, nil
}

// ProjectProjectItem processes an individual Project Item. This involves:
//
// - querying for additional timeline items, if applicable
// - calculating the total upvotes based on the data retrieved
// - calling the update function to update the Project Item on GitHub
func ProcessProjectItem(gh *githubv4.Client, ctx context.Context, item ProjectItemEdgeFragment) (Update, error) {

	var update Update
	content := item.GetContent()

	if content.TimelineItems.HasNextPage {
		additionalItems, err := GetAdditionalTimelineItems(gh, ctx, item.Id, content.TimelineItems.EndCursor)
		if err != nil {
			return update, err
		}

		content.TimelineItems.Nodes = append(content.TimelineItems.Nodes, additionalItems...)
	}

	update = Update{
		Id:      item.Id,
		Upvotes: githubv4.NewFloat(githubv4.Float(content.Upvotes())),
		Cursor:  item.Cursor,
	}
	slog.Info("update prepared for node", "node_id", update.Id, "upvotes", *update.Upvotes, "cursor", update.Cursor)

	return update, nil
}

// GetAdditionalTimelineItems is used when a Project Item has additional pages of timeline events.
// It paginates as needed, returning a slice of any additional timeline items after the given cursor.
func GetAdditionalTimelineItems(gh *githubv4.Client, ctx context.Context, node githubv4.ID, cursor githubv4.String) ([]TimelineItem, error) {
	var t []TimelineItem
	var query ProjectItemQuery

	variables := map[string]interface{}{
		"nodeId":         node,
		"timelineCursor": cursor,
	}

	for {
		slog.Info("querying for additional timeline items", "node_id", node, "cursor", variables["timelineCursor"])
		if err := gh.Query(ctx, &query, variables); err != nil {
			return t, err
		}

		t = append(t, query.GetContent().TimelineItems.Nodes...)

		if !query.HasNextPage() {
			break
		}

		variables["timelineCursor"] = query.GetContent().TimelineItems.EndCursor
	}

	slog.Info("all timeline items collected for node", "node_id", node)
	return t, nil
}

func UpdateProjectItems(gh *githubv4.Client, ctx context.Context, projectId githubv4.ID, fieldId githubv4.String, updates []Update) error {

	var mutation struct {
		UpdateProjectItemV2FieldValue struct {
			ClientMutationId string
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}

	input := githubv4.UpdateProjectV2ItemFieldValueInput{
		ProjectID: projectId,
		FieldID:   fieldId,
	}

	for _, update := range updates {
		input.ItemID = update.Id
		input.Value = githubv4.ProjectV2FieldValue{Number: update.Upvotes}

		slog.Info("updating upvotes", "item_id", update.Id, "upvotes", *update.Upvotes)
		if err := gh.Mutate(ctx, &mutation, input, nil); err != nil {
			return err
		}

		viper.Set("CURSOR", string(update.Cursor))
	}

	return nil
}
