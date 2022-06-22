FROM golang:1.18.2 AS build

ARG SERVICE_NAME

WORKDIR /go/src
COPY . /go/src

RUN go get -d -v ./...

RUN --mount=type=cache,target=/root/.cache/go-build go build -o /${SERVICE_NAME} ./cmd/main
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /${SERVICE_NAME}-migrate ./cmd/migration

FROM gcr.io/distroless/base AS runtime

ARG SERVICE_NAME

WORKDIR /${SERVICE_NAME}

COPY --from=build /go/src/config ./config
COPY --from=build /go/src/release-please ./release-please
COPY --from=build /go/src/internal/db/migration ./internal/db/migration

COPY --from=build /${SERVICE_NAME}-migrate ./
COPY --from=build /${SERVICE_NAME} ./
