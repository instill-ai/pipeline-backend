FROM golang:1.17.2 AS build

WORKDIR /go/src
COPY . /go/src

RUN go get -d -v ./...

RUN --mount=type=cache,target=/root/.cache/go-build go build -o /pipeline-backend ./cmd/main
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /pipeline-backend-migrate ./cmd/migration

FROM gcr.io/distroless/base AS runtime

WORKDIR /pipeline-backend

COPY --from=build /pipeline-backend ./
COPY --from=build /pipeline-backend-migrate ./
COPY --from=build /go/src/configs ./configs
COPY --from=build /go/src/internal/db/migration ./internal/db/migration

EXPOSE 8080/tcp
ENTRYPOINT ["./pipeline-backend"]
