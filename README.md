# Pipeline-backend

## Development

Pre-requirements:

- Go v1.17 or later installed on your development machine

For first run, execute from the project folder:

```bash
make
```

## Docker

```bash
# Build images with BuildKit
# https://docs.docker.com/develop/develop-images/build_enhancements/
DOCKER_BUILDKIT=1 docker build -t instill/pipeline-backend:0.0.2-dev .
```

This UI is published periodically on DockerHub: <https://hub.docker.com/r/instill/pipeline-backend>

You can run this to work with VDP using the [Instill VDP docker-compose](https://github.com/instill-ai/vdp/blob/main/docker-compose.yml#L80).
