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

FROM alpine:3.16

RUN apk add poppler-utils wv tidyhtml libc6-compat
RUN apk add unrtf --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing/

# USER nobody:nogroup

ARG SERVICE_NAME

WORKDIR /${SERVICE_NAME}

COPY --from=build /src/config ./config
COPY --from=build /src/release-please ./release-please
COPY --from=build  /src/pkg/db/migration ./pkg/db/migration

COPY --from=build /${SERVICE_NAME}-migrate ./
COPY --from=build  /${SERVICE_NAME}-worker ./
COPY --from=build  /${SERVICE_NAME} ./
