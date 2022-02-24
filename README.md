# Pipeline-backend

The pipeline-backend manages all pipeline resources and talks to [Visual Data Preparation (VDP)](https://github.com/instill-ai/vdp) for pipeline orchestration.

## Development

Pre-requirements:

- Go v1.17 or later installed on your development machine

### Binary build

```bash
$ make
```

### Docker build

```bash
# Build images with BuildKit
$ DOCKER_BUILDKIT=1 docker build -t instill/pipeline-backend:dev .
```

The latest images will be published to Docker Hub [repository](https://hub.docker.com/r/instill/pipeline-backend) at release time.

## License

See the [LICENSE](./LICENSE) file for licensing information.
