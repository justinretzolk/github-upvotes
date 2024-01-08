package main

import (
	"context"
	"os"

	"github.com/shurcooL/githubv4"
)

// A GitHubProject represents the GitHub project that will be read and updated
type GitHubProject struct {
	Organization         githubv4.String
	ProjectNumber        githubv4.Int
	ProjectId            githubv4.ID
	UpvotesFieldId       githubv4.String
	UpvotesCursorFieldId githubv4.String
	ItemsQueryCursor     githubv4.String
}

// NewProject returns a populated cursor, or an error if one is received
func NewGitHubProject(gh *githubv4.Client, org string, project int) (*GitHubProject, error) {

	ghp := &GitHubProject{
		Organization:  githubv4.String(org),
		ProjectNumber: githubv4.Int(project),
	}

	var query struct {
		Organization struct {
			Project struct {
				Id           githubv4.ID
				UpvotesField struct {
					Field struct {
						Id githubv4.String
					} `graphql:"...on ProjectV2Field"`
				} `graphql:"upvotes: field(name:\"Upvotes\")"` // todo: reconsider opinionated field name
				UpvotesCursorField struct {
					Field struct {
						Id githubv4.String
					} `graphql:"...on ProjectV2Field"`
				} `graphql:"cursor: field(name:\"Upvotes_Cursor\")"` // todo: reconsider opinionated field name
			} `graphql:"projectV2(number: $project)"`
		} `graphql:"organization(login: $organization)"`
	}

	vars := map[string]interface{}{
		"organization": ghp.Organization,
		"project":      ghp.ProjectNumber,
	}

	err := gh.Query(context.TODO(), &query, vars)
	if err != nil {
		return ghp, err
	}

	ghp.ProjectId = query.Organization.Project.Id
	ghp.UpvotesFieldId = query.Organization.Project.UpvotesField.Field.Id
	ghp.UpvotesCursorFieldId = query.Organization.Project.UpvotesCursorField.Field.Id

	// Optionally populate the cursor
	if c, ok := os.LookupEnv("CURSOR"); ok {
		ghp.ItemsQueryCursor = githubv4.String(c)
	}

	return ghp, nil
}

func (p *GitHubProject) UpdateUpvotes(gh *githubv4.Client, cursor *githubv4.String) error {

	// for pages of the list of project items
	for {
		items, err := GetProjectItems(gh, p.ProjectId, cursor)
		if err != nil {
			return err
		}

		// process each project item
		if cursor, err = ProcessProjectItems(gh, items.Edges); err != nil {
			return err
		}

		if !items.HasNextPage {
			break
		}

		cursor = &items.EndCursor
	}

	return nil
}
