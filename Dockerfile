FROM python:3.10-slim AS python-build

RUN python3.10 -m venv /opt/venv && \
    /opt/venv/bin/pip install --upgrade pip && \
    /opt/venv/bin/pip install pdfplumber mistral-common tokenizers transformers torch

FROM --platform=$TARGETPLATFORM golang:1.22.5 AS build

RUN apt update && apt install libtesseract-dev -y

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go get github.com/otiai10/gosseract/v2

ARG SERVICE_NAME TARGETOS TARGETARCH

RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=1 go build -tags=ocr -o /${SERVICE_NAME} ./cmd/main
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=1 go build -tags=ocr -o /${SERVICE_NAME}-worker ./cmd/worker
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-migrate ./cmd/migration
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-init ./cmd/init

FROM alpine:3.16

RUN apk add --no-cache \
    curl \
    poppler-utils \
    wv \
    tidyhtml \
    libc6-compat \
    tesseract-ocr \
    python3 \
    py3-pip \
    build-base \
    python3-dev \
    libffi-dev \
    libreoffice \
    msttcorefonts-installer \
    font-noto \
    font-noto-cjk \
    ffmpeg \
    && update-ms-fonts \
    && fc-cache -f \
    && python3 -m venv /opt/venv \
    && /opt/venv/bin/pip install --upgrade pip \
    && /opt/venv/bin/pip install pdfplumber tokenizers transformers \
    # mistral-common torch \
    && rm -rf /var/cache/apk/* /var/cache/fontconfig/*

# COPY --from=python-build /opt/venv/lib/python3.10/site-packages /opt/venv/lib/python3.10/site-packages

ARG TARGETARCH
ARG BUILDARCH
RUN apk add unrtf --repository=http://dl-cdn.alpinelinux.org/alpine/edge/community

USER nobody:nogroup

ARG SERVICE_NAME

WORKDIR /${SERVICE_NAME}

COPY --from=build --chown=nobody:nogroup /src/config ./config
COPY --from=build --chown=nobody:nogroup /src/release-please ./release-please
COPY --from=build --chown=nobody:nogroup /src/pkg/db/migration ./pkg/db/migration

COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-migrate ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-init ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-worker ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME} ./
