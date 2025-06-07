package github

import "github.com/shurcooL/githubv4"

type ViewerQuery struct {
	Viewer struct {
		Login     githubv4.String
		CreatedAt githubv4.DateTime
	}
}
