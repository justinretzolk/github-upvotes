package main

import (
	"testing"
)

func TestTimeLineItemUpvotes(t *testing.T) {
	testcases := []struct {
		want int
		tli  TimeLineItem
	}{
		{
			want: 2,
			tli: TimeLineItem{
				Type: "ConnectedEvent",
				ConnectedEvent: ConnectedEvent{
					CombinedBaseFragment: CombinedBaseFragment{
						Type: "Issue",
						Issue: BaseFragment{
							Comments: TotalCountFragment{
								TotalCount: 1,
							},
						},
					},
				},
			},
		},
		{
			want: 2,
			tli: TimeLineItem{
				Type: "CrossReferencedEvent",
				CrossReferencedEvent: CrossReferencedEvent{
					ConnectedEvent: ConnectedEvent{
						CombinedBaseFragment: CombinedBaseFragment{
							Type: "Issue",
							Issue: BaseFragment{
								Comments: TotalCountFragment{
									TotalCount: 1,
								},
							},
						},
					},
				},
			},
		},
		{
			want: 2,
			tli: TimeLineItem{
				Type: "IssueComment",
				IssueComment: IssueComment{
					Reactions: TotalCountFragment{
						TotalCount: 1,
					},
				},
			},
		},
		{
			want: 2,
			tli: TimeLineItem{
				Type: "MarkedAsDuplicateEvent",
				MarkedAsDuplicateEvent: MarkedAsDuplicateEvent{
					CombinedBaseFragment: CombinedBaseFragment{
						Type: "Issue",
						Issue: BaseFragment{
							Comments: TotalCountFragment{
								TotalCount: 1,
							},
						},
					},
				},
			},
		},
		{
			want: 1,
			tli: TimeLineItem{
				Type: "ReferencedEvent",
			},
		},
		{
			want: 1,
			tli: TimeLineItem{
				Type: "SubscribedEvent",
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(string(testcase.tli.Type), func(t *testing.T) {
			if got := testcase.tli.Upvotes(); got != testcase.want {
				t.Errorf("got: %v, want: %v", got, testcase.want)
			}
		})
	}
}

func TestBaseUpvotes(t *testing.T) {
	testcases := []struct {
		name           string
		commentsCount  int
		reactionsCount int
		want           int
	}{
		{"NoUpvotes", 0, 0, 0},
		{"Commented", 1, 0, 1},
		{"Reacted", 0, 1, 1},
		{"CommentedAndReacted", 1, 1, 2},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			b := BaseFragment{
				Comments: TotalCountFragment{
					TotalCount: testcase.commentsCount,
				},
				Reactions: TotalCountFragment{
					TotalCount: testcase.reactionsCount,
				},
			}

			if got := b.BaseUpvotes(); got != testcase.want {
				t.Errorf("got: %v, want: %v", got, testcase.want)
			}
		})
	}
}

func TestCombinedBaseUpvotes(t *testing.T) {
	testcases := []string{"Issue", "PullRequest"}

	for _, testcase := range testcases {
		t.Run(testcase, func(t *testing.T) {
			c := CombinedBaseFragment{
				Type: testcase,
			}

			b := BaseFragment{
				Comments: TotalCountFragment{
					TotalCount: 1,
				},
				Reactions: TotalCountFragment{
					TotalCount: 1,
				},
			}

			switch testcase {
			case "Issue":
				c.Issue = b
			case "PullRequest":
				c.PullRequest = b
			}

			if got := c.CombinedBaseUpvotes(); got != 2 {
				t.Errorf("got: %v, want: 2", got)
			}
		})
	}
}
