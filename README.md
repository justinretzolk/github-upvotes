# GitHub Upvotes

## Overview

The goal of this project is to calculate the number of "upvotes" that the items in a GitHub Project have (draft items excluded), and save the result to a custom field within the Project. "Upvotes" is defined as the sum of:

* The number of reactions to the issue or pull request
* The number of comments on the issue or pull request
* The number of reactions to each of the comments on the issue or pull request

This is still very much a work in progress.

## Configuration

| Environment Variable | Flag        | Required | Default | Description |
| -------------------- | ----------- | -------- | ------- | ----------- |
| `GITHUB_CURSOR`      | `--cursor`  | No       | None    | The cursor at which to begin paginating through Project Items |
| `GITHUB_UPDATE`      | `--update`  | No       | `false` | Whether or not to update the Upvotes field of the Project with the results |
| `GITHUB_ORG`         | `--org`     | Yes      | N/a     | The organization that owns the Project |
| `GITHUB_PROJECT`     | `--project` | Yes      | N/a     | The number of the Project to query |
| `GITHUB_TOKEN`       | N/a         | Yes      | N/a     | The `project` scoped GitHub token to use for authentication |