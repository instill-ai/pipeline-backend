# pipeline-backend

`pipeline-backend` manages all pipeline resources within [Visual Data Preparation (VDP)](https://github.com/instill-ai/vdp) to streamline data from a source connector to models, and to a destination connector at the end.

## Local dev

On the local machine, clone `vdp` repository in your workspace, move to the repository folder, and launch all dependent microservices:
```
$ cd <your-workspace>
$ git clone https://github.com/instill-ai/vdp.git
$ cd vdp
$ make dev PROFILE=pipeline
```

Clone `pipeline-backend` repository in your workspace and move to the repository folder:
```
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

### Run the Temporal worker

```bash
$ docker exec -it pipeline-backend /bin/bash
$ go run ./cmd/worker
```

### Run the integration test

``` bash
$ docker exec -it pipeline-backend /bin/bash
$ make integration-test
```

### Stop the dev container

```bash
$ make stop
```

### CI/CD

The latest images will be published to Docker Hub [repository](https://hub.docker.com/r/instill/pipeline-backend) at release.

## License

See the [LICENSE](./LICENSE) file for licensing information.
