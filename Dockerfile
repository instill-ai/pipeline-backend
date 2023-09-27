ARG GOLANG_VERSION
FROM --platform=$BUILDPLATFORM golang:${GOLANG_VERSION} AS build

ARG SERVICE_NAME

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . .

# install dependencies for text extraction (refer https://github.com/sajari/docconv)
RUN apt-get install poppler-utils wv unrtf tidy

ARG TARGETOS TARGETARCH
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME} ./cmd/main
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-migrate ./cmd/migration
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-worker ./cmd/worker

FROM gcr.io/distroless/base-debian12:nonroot

USER nonroot:nonroot

ARG SERVICE_NAME

WORKDIR /${SERVICE_NAME}

COPY --from=build --chown=nonroot:nonroot /src/config ./config
COPY --from=build --chown=nonroot:nonroot /src/release-please ./release-please
COPY --from=build --chown=nonroot:nonroot /src/pkg/db/migration ./pkg/db/migration

COPY --from=build --chown=nonroot:nonroot /${SERVICE_NAME}-migrate ./
COPY --from=build --chown=nonroot:nonroot /${SERVICE_NAME}-worker ./
COPY --from=build --chown=nonroot:nonroot /${SERVICE_NAME} ./
