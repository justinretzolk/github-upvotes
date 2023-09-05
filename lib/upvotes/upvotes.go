package upvotes

import (
	"context"
	"log/slog"

	"github.com/shurcooL/githubv4"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type Calculator struct {
	Client *githubv4.Client
}

func NewCalculator() Calculator {
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: viper.GetString("token")})
	httpClient := oauth2.NewClient(context.Background(), src)
	return Calculator{
		Client: githubv4.NewClient(httpClient),
	}
}

// CalculateUpvotes iterates through a Project's Project Items, calculating the number of upvotes
// for each one. It returns an error if any are encountered.
func (c Calculator) CalculateUpvotes() error {

	if !viper.IsSet("field_id") || !viper.IsSet("project_id") {
		c.setIds()
	}

	for {
		hasNextPage, err := c.calculateProjectItemUpvotes()
		if err != nil {
			return err
		}

		if !hasNextPage {
			slog.Info("upvote calculation complete", "end_cursor", viper.GetString("cursor"))
			break
		}
	}

	return nil
}

// calculateProjectItemUpvotes calculates the upvotes for the next Project Item after the cursor. It returns
// a boolean indicating whether there are additional Project Items to query or an error if one is encountered
func (c Calculator) calculateProjectItemUpvotes() (bool, error) {
	var (
		query         UpvoteQuery
		upvotes       int
		projectItemID string
	)

	vars := map[string]interface{}{
		"org":                           githubv4.String(viper.GetString("org")),
		"project":                       githubv4.Int(viper.GetInt("project_number")),
		"projectItemsCursor":            githubv4.String(viper.GetString("cursor")),
		"fieldName":                     githubv4.String(viper.GetString("field_name")),
		"commentsCursor":                (*githubv4.String)(nil),
		"trackedInIssuesCursor":         (*githubv4.String)(nil),
		"trackedIssuesCursor":           (*githubv4.String)(nil),
		"closingIssuesReferencesCursor": (*githubv4.String)(nil),
	}

	for {
		err := c.Client.Query(context.Background(), &query, vars)
		if err != nil {
			return false, err
		}
		projectItemID = query.GetProjectItemId()

		upvotes += query.ProjectItemConnectionsUpvotes()

		if !query.ProjectItemHasNextPage() {
			break
		}

		cursors := query.ProjectItemConnectionCursors()
		for k, v := range cursors {
			vars[k] = githubv4.String(v)
		}
		slog.Info("continuing to next page of project item", "project_item_id", projectItemID)
	}

	// add number of reactions, comments to the total upvotes
	upvotes += query.ProjectItemReactionsCount() + query.ProjectItemCommentCount()
	slog.Info("upvotes calculated", "project_item_id", projectItemID, "total_upvotes", upvotes)

	if viper.GetBool("write") {
		if err := c.updateProjectItem(projectItemID, upvotes); err != nil {
			return false, err
		}
	}

	hasNextPage, cursor := query.HasNextPage()
	viper.Set("cursor", cursor)

	return hasNextPage, nil
}

// setIds sets the ID of the Project and the Project's field used that is used to track upvotes
func (c Calculator) setIds() error {
	vars := map[string]interface{}{
		"org":       githubv4.String(viper.GetString("org")),
		"project":   githubv4.Int(viper.GetInt("project_number")),
		"fieldName": githubv4.String(viper.GetString("field_name")),
	}

	var query struct {
		Organization struct {
			Project struct {
				Id    string `graphql:"id"`
				Field struct {
					Upvotes struct {
						Id string `graphql:"id"`
					} `graphql:"... on ProjectV2Field"`
				} `graphql:"field(name: $fieldName)"`
			} `graphql:"projectV2(number: $project)"`
		} `graphql:"organization(login: $org)"`
	}

	slog.Info("getting project id and upvote field id")
	err := c.Client.Query(context.Background(), &query, vars)
	if err != nil {
		return err
	}

	slog.Info("setting id fields", "field_id", query.Organization.Project.Field.Upvotes.Id, "project_id", query.Organization.Project.Id)
	viper.Set("field_id", query.Organization.Project.Field.Upvotes.Id)
	viper.Set("project_id", query.Organization.Project.Id)

	return nil
}

// updateProjectItem takes a string representing the Project Item's ID and an int representing the calculated
// number of upvotes and updates the Project Item's upvote field's value to the current count of upvotes
func (c Calculator) updateProjectItem(i string, u int) error {
	var mutation struct {
		UpdateProjectItemV2FieldValue struct {
			ClientMutationId string
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}

	input := githubv4.UpdateProjectV2ItemFieldValueInput{
		ProjectID: githubv4.String(viper.GetString("project_id")),
		ItemID:    githubv4.String(i),
		FieldID:   githubv4.String(viper.GetString("field_id")),
		Value: githubv4.ProjectV2FieldValue{
			Number: githubv4.NewFloat(githubv4.Float(u)),
		},
	}

	slog.Info("updating project item upvotes", "project_item_id", i)
	if err := c.Client.Mutate(context.Background(), &mutation, input, nil); err != nil {
		return err
	}
	slog.Info("project item updated", "project_item_id", i, "mutation_id", mutation.UpdateProjectItemV2FieldValue.ClientMutationId)

	return nil
}
