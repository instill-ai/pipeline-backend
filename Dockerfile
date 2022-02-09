FROM golang:1.17.2 AS build

WORKDIR /go/src
COPY . /go/src

RUN go get -d -v ./...

RUN --mount=type=cache,target=/root/.cache/go-build go build -o /pipeline-backend ./cmd/
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /pipeline-backend-migrate ./internal/db/migrations

FROM gcr.io/distroless/base AS runtime

ENV GIN_MODE=release
WORKDIR /pipeline-backend

COPY --from=build /pipeline-backend ./
COPY --from=build /pipeline-backend-migrate ./
COPY --from=build /go/src/configs ./configs
COPY --from=build /go/src/internal/db/migrations ./internal/db/migrations

EXPOSE 8080/tcp
ENTRYPOINT ["./pipeline-backend"]
