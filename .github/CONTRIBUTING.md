# Contributing Guidelines

We appreciate your contribution to this amazing project! Any form of engagement is welcome, including but not limiting to
- feature request
- documentation wording
- bug report
- roadmap suggestion
- ...and so on!

Please refer to the [community contributing section](https://github.com/instill-ai/community#contributing) for more details.

## Development and codebase contribution

Before delving into the details to come up with your first PR, please familiarise yourself with the project structure of [Instill Core](https://github.com/instill-ai/community#instill-core).

### Prerequisites

- [Instill VDP](https://github.com/instill-ai/vdp)

### Pre-commit hooks

check out `.pre-commit-config.yaml` for the set of hooks that we used

### Local development

On the local machine, clone `vdp` repository in your workspace, move to the repository folder, and launch all dependent microservices:
```bash
$ cd <your-workspace>
$ git clone https://github.com/instill-ai/vdp.git
$ cd vdp
$ make latest PROFILE=pipeline
```

Clone `pipeline-backend` repository in your workspace and move to the repository folder:
```bash
$ cd <your-workspace>
$ git clone https://github.com/instill-ai/pipeline-backend.git
$ cd pipeline-backend
```

### Build the dev image

```bash
$ make build
```

### Run the dev container

```bash
$ make dev
```

Now, you have the Go project set up in the container, in which you can compile and run the binaries together with the integration test in each container shell.

### Run the server

```bash
$ docker exec -it pipeline-backend /bin/bash
$ go run ./cmd/migration
$ go run ./cmd/main
```

### Run the temporal worker

```bash
$ docker exec -it pipeline-backend /bin/bash
$ go run ./cmd/worker
```

### Run the unit tests

```bash
$ make coverate DBTEST=true
```

The repository tests in `make coverage` run against a real database (in contrast
to a mocked one) in order to increase the confidence of the tests. `DBTEST=true`
will create and migrate a test database to keep these queries isolated from the
main DB. You can set the database host and name by overriding the `TEST_DBHOST`
and `TEST_DBNAME` values.

### Run the integration test

```bash
$ docker exec -it pipeline-backend /bin/bash
$ make integration-test API_GATEWAY_URL=api-gateway:8080 DB_HOST=pg-sql
```

`API_GATEWAY_URL` points to the `api-gateway` container and triggers the public
API tests. If this variable is empty, the private API tests will be run.

At the end of the tests, some SQL queries are run to clean up the data.
`DB_HOST` points to the database host so the SQL connection can be established.
If empty, tests will try to connect to `localhost:5432`.

```bash
$ make integration-test API_GATEWAY_URL=api-gateway:8080 DB_HOST=pg-sql
`
### Stop the dev container

```bash
$ make stop
```

### Remove the dev container

```bash
$ make rm
```

### Sending PRs

Please take these general guidelines into consideration when you are sending a PR:

1. **Fork the Repository:** Begin by forking the repository to your GitHub account.
2. **Create a New Branch:** Create a new branch to house your work. Use a clear and descriptive name, like `<your-github-username>/<what-your-pr-about>`.
3. **Make and Commit Changes:** Implement your changes and commit them. We encourage you to follow these best practices for commits to ensure an efficient review process:
   - Adhere to the [conventional commits guidelines](https://www.conventionalcommits.org/) for meaningful commit messages.
   - Follow the [7 rules of commit messages](https://chris.beams.io/posts/git-commit/) for well-structured and informative commits.
   - Rearrange commits to squash trivial changes together, if possible. Utilize [git rebase](http://gitready.com/advanced/2009/03/20/reorder-commits-with-rebase.html) for this purpose.
4. **Push to Your Branch:** Push your branch to your GitHub repository: `git push origin feat/<your-feature-name>`.
5. **Open a Pull Request:** Initiate a pull request to our repository. Our team will review your changes and collaborate with you on any necessary refinements.

When you are ready to send a PR, we recommend you to first open a `draft` one. This will trigger a bunch of `tests` [workflows](https://github.com/instill-ai/pipeline-backend/tree/main/.github/workflows) running a thorough test suite on multiple platforms. After the tests are done and passed, you can now mark the PR `open` to notify the codebase owners to review. We appreciate your endeavour to pass the integration test for your PR to make sure the sanity with respect to the entire scope of **Instill Core**.

### CI/CD

- **pull_request** to the `main` branch will trigger the **`Integration Test`** workflow running the integration test using the image built on the PR head branch.
- **push** to the `main` branch will trigger
  - the **`Integration Test`** workflow building and pushing the `:latest` image on the `main` branch, following by running the integration test, and
  - the **`Release Please`** workflow, which will create and update a PR with respect to the up-to-date `main` branch using [release-please-action](https://github.com/google-github-actions/release-please-action).

Once the release PR is merged to the `main` branch, the [release-please-action](https://github.com/google-github-actions/release-please-action) will tag and release a version correspondingly.

The images are pushed to Docker Hub [repository](https://hub.docker.com/r/instill/pipeline-backend).

## Last words

Your contributions make a difference. Let's build something amazing together!
