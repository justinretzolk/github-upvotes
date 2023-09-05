package main

// TODO
// - [x] Read GH token, Project info from env
// - [x] Iterate through the items, handle pagination to get totals
// - [x] Update the Project Item's Upvote count

// FUTURE
// - [x] Accept flags for Project info
// - [ ] Allow notification if there's a delta above a certain limit

import (
	"github.com/justinretzolk/github-upvotes/cmd"
)

func main() {
	cmd.Execute()
}
