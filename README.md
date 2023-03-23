# pipeline-backend

[![Integration Test](https://github.com/instill-ai/pipeline-backend/actions/workflows/integration-test.yml/badge.svg)](https://github.com/instill-ai/pipeline-backend/actions/workflows/integration-test.yml)

`pipeline-backend` manages all pipeline resources within [Versatile Data Pipeline (VDP)](https://github.com/instill-ai/vdp) to streamline data from a source connector to models, and to a destination connector at the end.

## Local dev

On the local machine, clone `vdp` repository in your workspace, move to the repository folder, and launch all dependent microservices:
```bash
$ cd <your-workspace>
$ git clone https://github.com/instill-ai/vdp.git
$ cd vdp
$ make dev PROFILE=pipeline
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

### Run the integration test

```bash
$ docker exec -it pipeline-backend /bin/bash
$ make integration-test
```

### Stop the dev container

```bash
$ make stop
```

### CI/CD

- **push** to the `main` branch will trigger
  - the **`Create Release Candidate PR`** workflow, which will create and keep a PR to the `rc` branch up-to-date with respect to the `main` branch using [create-pull-request](github.com/peter-evans/create-pull-request) (commit message contains `release` string will be skipped), and
  - the **`Release Please`** workflow, which will create and update a PR with respect to the up-to-date `main` branch using [release-please-action](https://github.com/google-github-actions/release-please-action).
- **pull_request** to the `rc` branch will trigger the **`Integration Test`** workflow, which will run the integration test using the `:latest` images of **all** components.
- **push** to the `rc` branch will trigger the **`Integration Test`** workflow, which will build the `:rc` image and run the integration test using the `:rc` image of all components.
- Once the release PR is merged to the `main` branch, the [release-please-action](https://github.com/google-github-actions/release-please-action) will tag and release a version correspondingly.

The images are published to Docker Hub [repository](https://hub.docker.com/r/instill/pipeline-backend) at each CI/CD step.

## License

See the [LICENSE](./LICENSE) file for licensing information.
