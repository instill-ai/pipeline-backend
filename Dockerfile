FROM golang:1.18.2 AS build

WORKDIR /go/src
COPY . /go/src

RUN go get -d -v ./...

RUN --mount=type=cache,target=/root/.cache/go-build go build -o /pipeline-backend ./cmd/main
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /pipeline-backend-migrate ./cmd/migration

FROM gcr.io/distroless/base AS runtime

WORKDIR /pipeline-backend

COPY --from=build /go/src/config ./config
COPY --from=build /go/src/release-please ./release-please
COPY --from=build /go/src/internal/db/migration ./internal/db/migration

COPY --from=build /pipeline-backend-migrate ./
COPY --from=build /pipeline-backend ./

ENTRYPOINT ["./pipeline-backend"]
