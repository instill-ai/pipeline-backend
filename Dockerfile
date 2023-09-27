ARG GOLANG_VERSION
FROM --platform=$BUILDPLATFORM golang:${GOLANG_VERSION} AS build

ARG SERVICE_NAME

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETOS TARGETARCH
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME} ./cmd/main
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-migrate ./cmd/migration
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-worker ./cmd/worker

FROM debian:latest AS dependencies

# install dependencies for text extraction (refer https://github.com/sajari/docconv)
RUN apt update
RUN apt install poppler-utils wv unrtf tidy -y
RUN apt clean

FROM gcr.io/distroless/base:nonroot

USER nonroot:nonroot

ARG SERVICE_NAME

WORKDIR /${SERVICE_NAME}

COPY --from=dependencies /usr/bin/* /usr/bin/*
COPY --from=dependencies /usr/lib/* /usr/lib/*
COPY --from=dependencies /usr/local/bin/* /usr/local/bin/*
COPY --from=dependencies /usr/local/lib/* /usr/local/lib/*

COPY --from=build --chown=nonroot:nonroot /src/config ./config
COPY --from=build --chown=nonroot:nonroot /src/release-please ./release-please
COPY --from=build --chown=nonroot:nonroot /src/pkg/db/migration ./pkg/db/migration

COPY --from=build --chown=nonroot:nonroot /${SERVICE_NAME}-migrate ./
COPY --from=build --chown=nonroot:nonroot /${SERVICE_NAME}-worker ./
COPY --from=build --chown=nonroot:nonroot /${SERVICE_NAME} ./