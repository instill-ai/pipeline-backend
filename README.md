# pipeline-backend

The pipeline-backend manages all pipeline resources working with [Visual Data Preparation (VDP)](https://github.com/instill-ai/vdp).

## Development

Pre-requirements:

- Go v1.18 or later installed on your development machine

### Binary build

```bash
$ go mod tidy
$ go build -o pipeline-backend ./cmd/
```

### Docker build

```bash
$ make build
```

The latest images will be published to Docker Hub [repository](https://hub.docker.com/r/instill/pipeline-backend) at release.

## License

See the [LICENSE](./LICENSE) file for licensing information.
