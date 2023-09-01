package upvotes

import (
	"testing"
)

var itemTypeTestCases = []string{
	itemTypeIssue,
	itemTypePullRequest,
}

func TestGetProjectItemId(t *testing.T) {
	p := ProjectItem{
		ProjectItemId: "test",
	}

	q := generateTestUpvoteQuery(p)

	if got := q.GetProjectItemId(); got != "test" {
		t.Errorf("got: %v, want: test", got)
	}
}

func TestProjectItemCommentCount(t *testing.T) {
	for _, testcase := range itemTypeTestCases {
		t.Run(testcase, func(t *testing.T) {

			comments := ConnectionWithReactables{
				TotalCount: 5,
			}

			var cf contentFragment
			switch testcase {
			case itemTypeIssue:
				cf = IssueContentFragment{
					Comments: comments,
				}
			case itemTypePullRequest:
				cf = PullRequestContentFragment{
					Comments: comments,
				}
			}

			q := generateFilledTestUpvoteQuery(testcase, cf)

			if got := q.ProjectItemCommentCount(); got != 5 {
				t.Errorf("got: %v, want: 5", got)
			}
		})
	}
}

func TestProjectItemConnectionsUpvotes(t *testing.T) {
	testcases := []struct {
		itemType string
		want     int
	}{
		{itemTypeIssue, 3},
		{itemTypePullRequest, 2},
	}

	for _, testcase := range testcases {
		t.Run(testcase.itemType, func(t *testing.T) {
			cwr := ConnectionWithReactables{
				Nodes: []Reactable{
					{
						Reactions: Reactions{
							TotalCount: 1,
						},
					},
				},
			}

			var cf contentFragment
			switch testcase.itemType {
			case itemTypeIssue:
				cf = IssueContentFragment{
					Comments:        cwr,
					TrackedInIssues: cwr,
					TrackedIssues:   cwr,
				}
			case itemTypePullRequest:
				cf = PullRequestContentFragment{
					Comments:                cwr,
					ClosingIssuesReferences: cwr,
				}
			}

			q := generateFilledTestUpvoteQuery(testcase.itemType, cf)

			if got := q.ProjectItemConnectionsUpvotes(); got != testcase.want {
				t.Errorf("got: %v, want: %v", got, testcase.want)
			}
		})
	}
}

// TestProjectItemConnectionCursors

// TestProjectItemReactionsCount
func TestProjectItemReactionsCount(t *testing.T) {
	for _, testcase := range itemTypeTestCases {
		t.Run(testcase, func(t *testing.T) {
			r := Reactions{
				TotalCount: 1,
			}

			var cf contentFragment
			switch testcase {
			case itemTypeIssue:
				cf = IssueContentFragment{
					Reactions: r,
				}
			case itemTypePullRequest:
				cf = PullRequestContentFragment{
					Reactions: r,
				}
			}

			q := generateFilledTestUpvoteQuery(testcase, cf)

			if got := q.ProjectItemReactionsCount(); got != 1 {
				t.Errorf("got: %v, want: 1", got)
			}
		})
	}

}

// TestProjectItemHasNextPage

// TestHasNextPage

func generateTestUpvoteQuery(p ProjectItem) UpvoteQuery {
	return UpvoteQuery{
		Organization: Organization{
			Project: Project{
				ProjectItems: ProjectItems{
					Nodes: []ProjectItem{
						p,
					},
				},
			},
		},
	}
}

func generateFilledTestUpvoteQuery(t string, cf contentFragment) UpvoteQuery {
	var c Content
	switch t {
	case itemTypeIssue:
		c.Issue = cf.(IssueContentFragment)
	case itemTypePullRequest:
		c.PullRequest = cf.(PullRequestContentFragment)
	}

	p := ProjectItem{
		Type:    t,
		Content: c,
	}

	return generateTestUpvoteQuery(p)
}
