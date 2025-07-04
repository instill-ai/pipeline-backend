# Contributing Guidelines

We appreciate your contribution to this amazing project! Any form of engagement
is welcome, including but not limiting to

- feature request
- documentation wording
- bug report
- roadmap suggestion
- ...and so on!

## Introduction

Before delving into the details to come up with your first PR, please
familiarize yourself with the project structure of 🔮 [**Instill
Core**](https://github.com/instill-ai/instill-core).

You can also have an overview of the main [concepts](../README.md#concepts) in
the Instill Core domain.

If you want to extend a [component](../pkg/component) or contribute with a new
one, you might want to check the [component contribution
guide](../pkg/component/CONTRIBUTING.md).

## Local development

### Environment setup

If you want to see your changes in action you'll need to build Instill Core locally.
First, launch the latest version of [**Instill Core**](https://github.com/instill-ai/instill-core) suite. Then, build and
launch the [**Instill Core Pipeline**](https://github.com/instill-ai/pipeline-backend)
backend with your local changes.

#### Launching 🔮 Instill Core suite

```shell
cd $MY_WORKSPACE
git clone https://github.com/instill-ai/instill-core && cd instill-core
make latest
```

#### Remove the containers to avoid conflicts

```shell
docker rm -f pipeline-backend pipeline-backend-worker
```

#### Building the development container

```shell
cd $MY_WORKSPACE
git clone https://github.com/instill-ai/pipeline-backend && cd pipeline-backend
make build-dev && make dev
```

Now, you have the Go project set up in the container where you can compile
and run the binaries together with the integration test in each container shell.

##### Injecting component secrets

Some components can be configured with global secrets. This has several
applications:

- By accepting a global API key, some components have a default setup. When
  the `setup` block is omitted in the recipe, this API key will be used.
- In order to connect to 3rd party vendors via OAuth, the application
  client ID and secret must be injected.

You can set the values of these global secrets in
[`.env.secrets.component`](./.env.secrets.component) before running the Docker container in
order to add a global configuration to your components.

#### Run the server and the Temporal worker

```shell
docker exec pipeline-backend go run ./cmd/migration
docker exec pipeline-backend go run ./cmd/init
docker exec -d pipeline-backend go run ./cmd/worker # run without -d in a separate terminal if you want to access the logs
docker exec pipeline-backend go run ./cmd/main
```

#### Run the unit tests

```shell
make coverage DBTEST=true
```

The repository tests in `make coverage` run against a real database (in contrast
to a mocked one) in order to increase the confidence of the tests. `DBTEST=true`
will create and migrate a test database to keep these queries isolated from the
main DB. You can set the database host and name by overriding the `TEST_DBHOST`
and `TEST_DBNAME` values.

Certain tests depend on external packages and aren't run by default:

- For [`docconv`](https://github.com/sajari/docconv) tests, add `OCR=true` flag and install its [dependencies](https://github.com/sajari/docconv?tab=readme-ov-file#dependencies).
- For [`onnxruntime`](https://github.com/microsoft/onnxruntime) tests, add `ONNX=true` flag. Follow the [guideline](./#set-up-onnx-runtime) to set up ONNX Runtime (Linux only).

#### Run the integration tests

```shell
docker exec -it pipeline-backend /bin/bash
make integration-test API_GATEWAY_URL=api-gateway:8080 DB_HOST=pg-sql
```

`API_GATEWAY_URL` points to the `api-gateway` container and triggers the public
API tests. If this variable is empty, the private API tests will be run.

At the end of the tests, some SQL queries are run to clean up the data.
`DB_HOST` points to the database host so the SQL connection can be established.
If empty, tests will try to connect to `localhost:5432`.

#### Remove the dev container

```shell
make rm
```

### Set up ONNX Runtime (Linux only)

1. Download the latest [ONNX Runtime release](https://github.com/microsoft/onnxruntime/releases) for your system.

2. Install ONNX Runtime:

```shell
sudo mkdir -p /usr/local/onnxruntime
sudo tar -xzf onnxruntime-*-*-*.tgz -C /usr/local/onnxruntime --strip-components=1
export ONNXRUNTIME_ROOT_PATH=/usr/local/onnxruntime
export LD_RUN_PATH=$ONNXRUNTIME_ROOT_PATH/lib
export LIBRARY_PATH=$ONNXRUNTIME_ROOT_PATH/lib
export C_INCLUDE_PATH=$ONNXRUNTIME_ROOT_PATH/include
```

**Note:** If you don't have sudo access, extract to a user-writeable location (e.g., `~/onnxruntime`), set `ONNXRUNTIME_ROOT_PATH` accordingly, and adjust the environment variables as shown above. No need to create symlinks in this case.

## Codebase contribution

### Pre-commit hooks

Check out `.pre-commit-config.yaml` for the set of hooks that we use.

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
