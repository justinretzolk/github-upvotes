# [WIP] GitHub Upvotes

This project is meant to calculate "upvotes" for items in a GitHub Project, then update a field in the Project with the result. It's meant to eventually be run in a GitHub Action The README is a WIP.

### Arguments

| Flag    | Required | Environment Variable | Description                                                             |
| ------- | -------- | -------------------- | ----------------------------------------------------------------------- |
| token   | true     | GITHUB_TOKEN         | The token used to authenticate with the GitHub API                      |
| project | true     | PROJECT_ID           | The ID of the GitHub Project to be updated                              |
| field   | true     | FIELD_ID             | The ID of the Field in the GitHub Project that is used to track upvotes |
| verbose | false    | RUNNER_DEBUG         | Output verbose / debug logging                                          |

### Retrieving Required IDs

The `gh` CLI can be used to retrieve the Project and Field IDs.

Project ID:

```shell
$ gh project view $PROJECT_NUMBER --owner $ORGANIZATION_OR_USERNAME --format json --jq '.id'
```

Field ID (replace `<name_of_field>` with the name of the field used to track upvotes)

```shell
$ gh project field-list 231 --owner hashicorp --format json --jq '.fields[] | select(.name == "<name_of_field>") | .id'
```
