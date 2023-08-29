# GitHub Upvotes

## Overview

The goal of this project is to calculate the number of "upvotes" that the items in a GitHub Project have (draft items excluded), and save the result to a custom field within the Project. "Upvotes" is defined as the sum of:

* The number of reactions to the issue or pull request
* The number of comments on the issue or pull request
* The number of reactions to each of the comments on the issue or pull request
* The number of reactions to linked issues/PRs

## Goals

- [ ] Ability to run from the CLI
- [ ] Ability to run in Docker
- [ ] Ability to run as a GitHub Action