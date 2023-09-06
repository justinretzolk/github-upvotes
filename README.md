# GitHub Upvotes

## Overview

The goal of this project is to calculate the number of "upvotes" that the Project Items in a GitHub Project have (draft, closed, and archived items excluded), and save the result to a custom field within the Project. "Upvotes" is defined as the sum of:

* Total reactions
* Number of comments
* Total reactions to each of the comments
* Number of linked issues
* Total reactions to each of the linked issues

## Configuration

The program has several arguments that may be supplied either by flags or environment variables

| Flag               | Environment Variable    | Required | Default | Description                                                                   |
| ------------------ | ----------------------- | -------- | ------- | ----------------------------------------------------------------------------- |
| `--org`            | `GITHUB_ORG`            | Yes      | None    | The organization that owns the Project                                        |
| `--project_number` | `GITHUB_PROJECT_NUMBER` | Yes      | None    | The number of the Project                                                     |
| `--field_name`     | `GITHUB_FIELD_NAME`     | Yes      | None    | The name of the Project's Field that is used to track upvotes                 |
| None               | `GITHUB_TOKEN`          | Yes      | None    | The token used to authenticate with the GitHub                                |
| `--cursor`         | `GITHUB_CURSOR`         | No       | None    | The cursor of the Project Item to start after                                 |
| `--field_id`       | `GITHUB_FIELD_ID`       | No       | None    | The ID of the Project's Field that is used to track upvotes                   |
| `--project_id`     | `GITHUB_PROJECT_ID`     | No       | None    | The ID of the Project                                                         |
| `--write`          | `GITHUB_WRITE`          | No       | `false` | Whether or not to write the results to the Project (if `false`, is a dry run) |
