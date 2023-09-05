package upvotes

import (
	"reflect"
	"testing"
)

var itemTypeTestCases = []string{
	itemTypeIssue,
	itemTypePullRequest,
}

const (
	endCursor = "endcursor1234567890"
)

func TestGetProjectItemId(t *testing.T) {
	p := ProjectItem{
		Id: "test",
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
					CommonContentFragment: CommonContentFragment{
						Comments: comments,
					},
				}
			case itemTypePullRequest:
				cf = PullRequestContentFragment{
					CommonContentFragment: CommonContentFragment{
						Comments: comments,
					},
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
					CommonContentFragment: CommonContentFragment{
						Comments: cwr,
					},
					TrackedInIssues: cwr,
					TrackedIssues:   cwr,
				}
			case itemTypePullRequest:
				cf = PullRequestContentFragment{
					CommonContentFragment: CommonContentFragment{
						Comments: cwr,
					},
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

func TestProjectItemConnectionCursors(t *testing.T) {
	testcases := []struct {
		itemType string
		want     map[string]string
	}{
		{itemTypeIssue, map[string]string{"commentsCursor": endCursor, "trackedInIssuesCursor": endCursor, "trackedIssuesCursor": endCursor}},
		{itemTypePullRequest, map[string]string{"commentsCursor": endCursor, "closingIssueReferencesCursor": endCursor}},
	}

	c := ConnectionWithReactables{
		PageInfo: PageInfo{
			EndCursor: endCursor,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.itemType, func(t *testing.T) {
			var cf contentFragment
			switch testcase.itemType {
			case itemTypeIssue:
				cf = IssueContentFragment{
					CommonContentFragment: CommonContentFragment{
						Comments: c,
					},
					TrackedIssues:   c,
					TrackedInIssues: c,
				}
			case itemTypePullRequest:
				cf = PullRequestContentFragment{
					CommonContentFragment: CommonContentFragment{
						Comments: c,
					},
					ClosingIssuesReferences: c,
				}
			}

			q := generateFilledTestUpvoteQuery(testcase.itemType, cf)

			if got := q.ProjectItemConnectionCursors(); !reflect.DeepEqual(got, testcase.want) {
				t.Errorf("got: %v, want: %v", got, testcase.want)
			}
		})
	}
}

func TestProjectItemReactionsCount(t *testing.T) {
	r := Reactions{
		TotalCount: 1,
	}

	for _, testcase := range itemTypeTestCases {
		t.Run(testcase, func(t *testing.T) {

			var cf contentFragment
			switch testcase {
			case itemTypeIssue:
				cf = IssueContentFragment{
					CommonContentFragment: CommonContentFragment{
						Reactions: r,
					},
				}
			case itemTypePullRequest:
				cf = PullRequestContentFragment{
					CommonContentFragment: CommonContentFragment{
						Reactions: r,
					},
				}
			}

			q := generateFilledTestUpvoteQuery(testcase, cf)

			if got := q.ProjectItemReactionsCount(); got != 1 {
				t.Errorf("got: %v, want: 1", got)
			}
		})
	}
}

func TestProjectItemHasNextPage(t *testing.T) {
	testcases := []struct {
		name        string
		itemType    string
		want        bool
		comments    bool
		tracked     bool
		trackedIn   bool
		closingRefs bool
	}{
		{"IssueNoMorePages", itemTypeIssue, false, false, false, false, false},
		{"IssueMoreComments", itemTypeIssue, true, true, false, false, false},
		{"IssueMoreTrackedIssues", itemTypeIssue, true, false, true, false, false},
		{"IssueMoreTrackedInIssues", itemTypeIssue, true, false, false, true, false},
		{"PullRequestNoMorePages", itemTypePullRequest, false, false, false, false, false},
		{"PullRequestMoreComments", itemTypePullRequest, true, true, false, false, false},
		{"PullRequestMoreClosingIssueReferences", itemTypePullRequest, true, false, false, false, true},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			var cf contentFragment
			switch testcase.itemType {
			case itemTypeIssue:
				cf = IssueContentFragment{
					CommonContentFragment: CommonContentFragment{
						Comments: ConnectionWithReactables{
							PageInfo: PageInfo{
								HasNextPage: testcase.comments,
							},
						},
					},
					TrackedIssues: ConnectionWithReactables{
						PageInfo: PageInfo{
							HasNextPage: testcase.tracked,
						},
					},
					TrackedInIssues: ConnectionWithReactables{
						PageInfo: PageInfo{
							HasNextPage: testcase.trackedIn,
						},
					},
				}
			case itemTypePullRequest:
				cf = PullRequestContentFragment{
					CommonContentFragment: CommonContentFragment{
						Comments: ConnectionWithReactables{
							PageInfo: PageInfo{
								HasNextPage: testcase.comments,
							},
						},
					},
					ClosingIssuesReferences: ConnectionWithReactables{
						PageInfo: PageInfo{
							HasNextPage: testcase.closingRefs,
						},
					},
				}
			}

			q := generateFilledTestUpvoteQuery(testcase.itemType, cf)

			if got := q.ProjectItemHasNextPage(); got != testcase.want {
				t.Errorf("got: %v, want %v", got, testcase.want)
			}
		})
	}
}

func TestHasNextPage(t *testing.T) {
	q := UpvoteQuery{
		Organization: Organization{
			Project: Project{
				ProjectItems: ProjectItems{
					PageInfo: PageInfo{
						HasNextPage: true,
						EndCursor:   endCursor,
					},
				},
			},
		},
	}

	if h, c := q.HasNextPage(); !h || c != endCursor {
		t.Errorf("got: %v, %v, want: true, %v", h, c, endCursor)
	}
}

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
