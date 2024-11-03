ARG GOLANG_VERSION=1.22.5
FROM golang:${GOLANG_VERSION}-bullseye AS build

ARG TARGETOS TARGETARCH K6_VERSION XK6_VERSION

RUN apt-get update && apt-get install -y \
    build-essential \
    libleptonica-dev \
    libtesseract-dev \
    libsoxr-dev \
    && rm -rf /var/lib/apt/lists/*

# Install ONNX Runtime (latest release)
ENV ONNXRUNTIME_ROOT_PATH=/usr/local/onnxruntime
RUN apt update && \
    apt install -y wget jq && \
    LATEST_VERSION=$(wget -qO- https://api.github.com/repos/microsoft/onnxruntime/releases/latest | jq -r .tag_name) && \
    ONNX_ARCH=$([ "$TARGETARCH" = "arm64" ] && echo "aarch64" || echo "x64") && \
    wget https://github.com/microsoft/onnxruntime/releases/download/${LATEST_VERSION}/onnxruntime-linux-${ONNX_ARCH}-${LATEST_VERSION#v}.tgz && \
    tar -xzf onnxruntime-linux-${ONNX_ARCH}-${LATEST_VERSION#v}.tgz && \
    mv onnxruntime-linux-${ONNX_ARCH}-${LATEST_VERSION#v} ${ONNXRUNTIME_ROOT_PATH} && \
    rm onnxruntime-linux-${ONNX_ARCH}-${LATEST_VERSION#v}.tgz && \
    apt remove -y wget jq && \
    apt autoremove -y && \
    rm -rf /var/lib/apt/lists/*

# Set environment variables and create symlinks for ONNX Runtime
ENV C_INCLUDE_PATH=${ONNXRUNTIME_ROOT_PATH}/include
ENV LD_RUN_PATH=${ONNXRUNTIME_ROOT_PATH}/lib
ENV LIBRARY_PATH=${ONNXRUNTIME_ROOT_PATH}/lib

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG SERVICE_NAME TARGETOS TARGETARCH
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -tags=ocr,onnx -o /${SERVICE_NAME} ./cmd/main
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-worker ./cmd/worker
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-migrate ./cmd/migration
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-init ./cmd/init

FROM debian:bullseye-slim

# Install Python, create virtual environment, and install pdfplumber
RUN apt update && \
    apt install -y curl python3 python3-venv poppler-utils wv unrtf tidy tesseract-ocr libtesseract-dev libreoffice ffmpeg libsoxr-dev chromium qpdf && \
    python3 -m venv /opt/venv && \
    /opt/venv/bin/pip install pdfplumber mistral-common tokenizers && \
    rm -rf /var/lib/apt/lists/*

# copy ONNX runtime from build stage
ENV ONNXRUNTIME_ROOT_PATH=/usr/local/onnxruntime
COPY --from=build --chown=nobody:nogroup /usr/local/onnxruntime ${ONNXRUNTIME_ROOT_PATH}

# Set environment variables and create symlinks for ONNX Runtime
ENV C_INCLUDE_PATH=${ONNXRUNTIME_ROOT_PATH}/include
RUN ln -s ${ONNXRUNTIME_ROOT_PATH}/lib/libonnxruntime.so* /usr/lib/

USER nobody:nogroup

ARG SERVICE_NAME

WORKDIR /${SERVICE_NAME}

ENV GODEBUG=tlsrsakex=1

COPY --from=build --chown=nobody:nogroup /src/config ./config
COPY --from=build --chown=nobody:nogroup /src/release-please ./release-please
COPY --from=build --chown=nobody:nogroup /src/pkg/db/migration ./pkg/db/migration

COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-migrate ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-init ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-worker ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME} ./

# Set up ONNX model and environment variable
COPY --chown=nobody:nogroup ./pkg/component/resources/onnx/silero_vad.onnx /${SERVICE_NAME}/pkg/component/resources/onnx/silero_vad.onnx
ENV ONNX_MODEL_FOLDER_PATH=/${SERVICE_NAME}/pkg/component/resources/onnx
