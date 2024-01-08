package main

import (
	"context"
	"log/slog"

	"github.com/shurcooL/githubv4"
)

func GetProjectItems(gh *githubv4.Client, id githubv4.ID, cursor *githubv4.String) (ProjectItemsFragment, error) {
	var query ProjectItemsQuery
	items := query.Items

	variables := map[string]interface{}{
		"nodeId":         id,
		"cursor":         cursor,
		"timelineCursor": (*githubv4.String)(nil), // not used here, but a required variable nonetheless
	}

	err := gh.Query(context.TODO(), &query, variables)

	return items, err
}

func ProcessProjectItems(gh *githubv4.Client, items []ProjectItemEdgeFragment) (githubv4.String, error) {

	var cursor githubv4.String

	for _, item := range items {
		if item.Skip() {
			slog.Info("item skipped", "item_id", item.Id)
			cursor = item.Cursor
			continue
		}

		if err := ProcessProjectItem(gh, item); err != nil {
			return cursor, err
		}

		cursor = item.Cursor
	}

	return cursor, nil
}

func ProcessProjectItem(gh *githubv4.Client, item ProjectItemEdgeFragment) error {

	content := item.GetContent()

	if content.TimelineItems.HasNextPage {
		slog.Info("timeline item has additional page(s)", "timeline_item", item.Id)

		additionalItems, err := GetAdditionalTimelineItems(gh, item.Id, content.TimelineItems.EndCursor)
		if err != nil {
			return err
		}

		content.TimelineItems.Nodes = append(content.TimelineItems.Nodes, additionalItems...)
	}

	upvotes := content.Upvotes()

	// add the previously known upvotes
	// todo: the delta could be calculated and used for alerting
	upvotes += int(item.UpvotesField.Value)

	// update the project item
	slog.Info("upvotes calculated", "item_id", item.Id, "upvotes", upvotes)

	return nil
}

func GetAdditionalTimelineItems(gh *githubv4.Client, node githubv4.ID, cursor githubv4.String) ([]TimelineItem, error) {
	var t []TimelineItem
	var query ProjectItemQuery

	variables := map[string]interface{}{
		"nodeId":         node,
		"timelineCursor": cursor,
	}

	for {
		slog.Info("querying for additional timeline items", "node_id", node, "cursor", variables["timelineCursor"])
		if err := gh.Query(context.TODO(), &query, variables); err != nil {
			return t, err
		}

		t = append(t, query.GetContent().TimelineItems.Nodes...)

		if !query.HasNextPage() {
			break
		}

		variables["timelineCursor"] = query.GetContent().TimelineItems.EndCursor
	}

	return t, nil
}
