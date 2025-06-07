package update

import (
	"context"
	"fmt"

	"github.com/justinretzolk/github-upvotes/pkg/github"
)

func UpdateUpvotes(token string) error {
	// Get GitHub Client
	client := github.NewClient(token)

	// Calculate Upvotes
	var viewer github.ViewerQuery
	err := client.Query(context.TODO(), &viewer, nil)
	if err != nil {
		return err
	}

	fmt.Println("Viewer: " + viewer.Viewer.Login)
	// Update Project

	return nil
}
