package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type Client struct {
	Conn              *githubv4.Client
	Organization      string
	Project           int
	ProjectItemCursor *string
}

// NewClient generates a new Client
func NewClient() (*Client, error) {

	// this is the only opportunity for failure, so try it first
	proj, err := strconv.Atoi(os.Getenv("GITHUB_PROJECT_NUMBER"))
	if err != nil {
		return nil, fmt.Errorf("unable to convert project number to int: %v", err)
	}

	// token source for GitHub client
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)

	client := &Client{
		Conn:         githubv4.NewClient(oauth2.NewClient(context.Background(), src)),
		Organization: os.Getenv("GITHUB_ORGANIZATION"),
		Project:      proj,
	}

	if cursor, ok := os.LookupEnv("PROJECT_ITEM_CURSOR"); ok {
		client.ProjectItemCursor = &cursor
	}

	return client, nil
}

// SetProjectItemCursor updates the Client's ProjectItemCursor
func (c *Client) SetProjectItemCursor(cursor string) {
	c.ProjectItemCursor = &cursor
}
