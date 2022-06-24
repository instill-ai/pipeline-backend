FROM --platform=$BUILDPLATFORM golang:1.18.2 AS build

ARG SERVICE_NAME

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETOS TARGETARCH
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME} ./cmd/main
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-migrate ./cmd/migration

FROM gcr.io/distroless/base

ARG SERVICE_NAME

WORKDIR /${SERVICE_NAME}

COPY --from=build /src/config ./config
COPY --from=build /src/release-please ./release-please
COPY --from=build /src/internal/db/migration ./internal/db/migration

COPY --from=build /${SERVICE_NAME}-migrate ./
COPY --from=build /${SERVICE_NAME} ./
