package main

import (
	"context"
	"log/slog"
	"sync"

	"github.com/shurcooL/githubv4"
)

// GetProjectItems pages through the list of items within the GitHub Project. It requires a context, GitHub client,
// the ID of the GitHub Project, and a channel on which to send errors. It returns a channel that receives ProjectItemEdgeFragment
// types, and a WaitGroup used for synchronizing when the next page should be queried.
func GetProjectItems(ctx context.Context, gh *githubv4.Client, projectId githubv4.ID, errChan chan<- error) (<-chan ProjectItemEdgeFragment, *sync.WaitGroup) {
	out := make(chan ProjectItemEdgeFragment)
	var wg sync.WaitGroup

	var query ProjectItemsQuery
	variables := map[string]interface{}{
		"nodeId": projectId,
		"cursor": (*githubv4.String)(nil),

		// TODO: Fix this
		// not used here, but a required variable nonetheless
		"timelineCursor": (*githubv4.String)(nil),
	}

	go func() {
	pager:
		for {
			// paginated query, errors should cancel the context, need error channel as input
			if err := gh.Query(ctx, &query, variables); err != nil {
				// send the error to the channel so that the context gets cancelled,
				// break the for loop so that the channel gets closed
				errChan <- err
				break
			}

			// work through the project items to see which ones should be skipped
			for _, item := range query.Items.Edges {
				if !item.Skip() {
					wg.Add(1)
					out <- item
				}
			}

			// wait on waitgroup, context to be cancelled
			wg.Wait()
			select {
			case <-ctx.Done():
				break pager
			default:
				if !query.HasNextPage() {
					break pager
				}

				// update the cursor before breaking the select and moving to the next iteration
				variables["cursor"] = query.Items.EndCursor
				break
			}
		}
		close(out)
	}()

	return out, &wg
}

// ProcessProjectItems processing incoming ProjectItemEdgeFragment types, calculates the number of upvotes, and
// generates an Update type, representing the data required to update a project item's upvotes. It requires a context,
// GitHub client, a channel in which to receive ProjectItemEdgeFragment types, and a channel on which to report errors.
// It returns a channel that receives Update types.
func ProcessProjectItems(ctx context.Context, gh *githubv4.Client, in <-chan ProjectItemEdgeFragment, errChan chan<- error) <-chan Update {
	out := make(chan Update)

	process := func(item ProjectItemEdgeFragment) {
		content := item.GetContent()

		if content.TimelineItems.HasNextPage {
			var query ProjectItemQuery

			variables := map[string]interface{}{
				"nodeId":         item.Id,
				"timelineCursor": content.TimelineItems.EndCursor,
			}

			for {
				slog.Debug("querying for additional timeline items", "node_id", item.Id)
				if err := gh.Query(ctx, &query, variables); err != nil {
					errChan <- err
					return
				}

				content.TimelineItems.Nodes = append(content.TimelineItems.Nodes, query.GetContent().TimelineItems.Nodes...)

				if !query.HasNextPage() {
					break
				}

				variables["timelineCursor"] = query.GetContent().TimelineItems.EndCursor
			}
		}

		out <- Update{
			Id:      item.Id,
			Upvotes: githubv4.NewFloat(githubv4.Float(content.Upvotes())),
			Cursor:  item.Cursor,
		}
	}

	go func() {
		for item := range in {
			go process(item)
		}
		close(out)
	}()

	return out
}

// UpdateProjectItems processes incoming Update types and uses them to update the project item's upvote count.
// It requires a context, GitHub client, a WaitGroup for syncronizing pagination, the GitHub Project's ID,
// and the ID of the custom 'upvotes' field on the Project. It returns a channel used to indicate that all
// updates have completed.
func UpdateProjectItems(ctx context.Context, gh *githubv4.Client, wg *sync.WaitGroup, projectId githubv4.ID, fieldId githubv4.ID, in <-chan Update, errChan chan<- error) <-chan struct{} {
	out := make(chan struct{})

	var mutation struct {
		UpdateProjectItemV2FieldValue struct {
			ClientMutationId string
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}

	input := githubv4.UpdateProjectV2ItemFieldValueInput{
		ProjectID: projectId,
		FieldID:   fieldId,
	}

	go func() {
		for update := range in {

			input.ItemID = update.Id
			input.Value = githubv4.ProjectV2FieldValue{Number: update.Upvotes}

			if err := gh.Mutate(ctx, &mutation, input, nil); err != nil {
				errChan <- err
				break
			}

			wg.Done()
			slog.Info("updated project item", "item_id", update.Id, "upvotes", *update.Upvotes)
		}
		close(out)
	}()

	return out
}
