# [WIP] GitHub Upvotes

## Overview

The goal of this project is to calculate the number of "upvotes" that the Project Items in a GitHub Project have (draft, closed, and archived items excluded), and save the result to a custom field within the Project. "Upvotes" is defined as the sum of:

* Total reactions to the item
* Number of comments on the item

In addition, upvotes are added for the following [timeline items](https://docs.github.com/en/graphql/reference/enums#issuetimelineitemsitemtype):

* `CONNECTED_EVENT` (+ number of reactions and comments on the connected item)
* `CROSS_REFERENCED_EVENT` (+ number of reactions and comments on the source item)
* `ISSUE_COMMENT` (+ number of reactions to the comment)
* `MARKED_AS_DUPLICATE_EVENT` (+ number of reactions and comments on the canonical item)
* `REFERENCED_EVENT`
* `SUBSCRIBED_EVENT`

## Configuration

The program has several arguments that may be supplied by environment variables

| Environment Variable    | Required | Description                                                                   |
| ----------------------- | -------- | ----------------------------------------------------------------------------- |
| `GITHUB_ORGANIZATION`   | Yes      | The organization that owns the Project                                        |
| `PROJECT_NUMBER`        | Yes      | The number of the Project                                                     |
| `UPVOTE_FIELD_NAME`     | Yes      | The name of the Project's Field that is used to track upvotes                 |
| `GITHUB_TOKEN`          | Yes      | The token used to authenticate with the GitHub                                |
| `GITHUB_OUTPUT`         | Yes      | Used to write output for GitHub Actions                                       |
| `CURSOR`                | No       | The cursor of the Project Item to start after                                 |