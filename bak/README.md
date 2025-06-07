# [WIP] GitHub Upvotes

This project is meant to calculate "upvotes" for items in a GitHub Project, then update a field in the Project with the result. It's meant to eventually be run in a GitHub Action The README is a WIP.

Required environment variables:

- `GITHUB_TOKEN`: a token with permissions to read issues/prs in the repository + read/write to the project
- `GITHUB_PROJECT_ID`: the ID of the GitHub Project. 
- `GITHUB_FIELD_ID`: the ID of the 'upvotes' field in the GitHub Project.

For the project and field IDs respectively, see [here](https://cli.github.com/manual/gh_project_view) and [here](https://cli.github.com/manual/gh_project_field-list). 

Optional environment variables:

- `RUNNER_DEBUG`: matches GitHub's environment variable for Actions debugging.