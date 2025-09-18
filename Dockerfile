ARG GOLANG_VERSION=1.24.4

# ===============================
# Stage 1: Build dependencies
# ===============================
FROM golang:${GOLANG_VERSION}-bullseye AS build-deps

ARG TARGETOS TARGETARCH

# Install build dependencies in one layer
RUN apt-get update && apt-get install -y \
    build-essential \
    libleptonica-dev \
    libtesseract-dev \
    libsoxr-dev \
    wget \
    jq \
    && rm -rf /var/lib/apt/lists/*

# ===============================
# Stage 2: Binary dependencies
# ===============================
FROM build-deps AS binary-deps

ENV ONNXRUNTIME_ROOT_PATH=/usr/local/onnxruntime

# Download and install FFmpeg
RUN FFMPEG_ARCH=$([ "$TARGETARCH" = "arm64" ] && echo "arm64" || echo "amd64") && \
    (wget --timeout=30 --tries=3 https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-${FFMPEG_ARCH}-static.tar.xz || \
    (echo "Primary source failed, trying GitHub fallback..." && \
    if [ "$FFMPEG_ARCH" = "amd64" ]; then \
        wget https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linux64-gpl.tar.xz -O ffmpeg-release-${FFMPEG_ARCH}-static.tar.xz; \
    else \
        wget https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linuxarm64-gpl.tar.xz -O ffmpeg-release-${FFMPEG_ARCH}-static.tar.xz; \
    fi)) && \
    tar xvf ffmpeg-release-${FFMPEG_ARCH}-static.tar.xz && \
    if [ -d ffmpeg-*-static ]; then \
        mv ffmpeg-*-static/ffmpeg /usr/local/bin/ && \
        mv ffmpeg-*-static/ffprobe /usr/local/bin/; \
    else \
        mv ffmpeg-*/bin/ffmpeg /usr/local/bin/ && \
        mv ffmpeg-*/bin/ffprobe /usr/local/bin/; \
    fi && \
    rm -rf ffmpeg-*

# Download and install ONNX Runtime
RUN ONNX_ARCH=$([ "$TARGETARCH" = "arm64" ] && echo "aarch64" || echo "x64") && \
    VERSION=v1.20.1 && \
    wget https://github.com/microsoft/onnxruntime/releases/download/${VERSION}/onnxruntime-linux-${ONNX_ARCH}-${VERSION#v}.tgz && \
    tar -xzf onnxruntime-linux-${ONNX_ARCH}-${VERSION#v}.tgz && \
    mv onnxruntime-linux-${ONNX_ARCH}-${VERSION#v} ${ONNXRUNTIME_ROOT_PATH} && \
    rm onnxruntime-linux-${ONNX_ARCH}-${VERSION#v}.tgz && \
    apt remove -y wget jq && \
    apt autoremove -y && \
    rm -rf /var/lib/apt/lists/*

# Set environment variables for ONNX Runtime
ENV C_INCLUDE_PATH=${ONNXRUNTIME_ROOT_PATH}/include
ENV LD_RUN_PATH=${ONNXRUNTIME_ROOT_PATH}/lib
ENV LIBRARY_PATH=${ONNXRUNTIME_ROOT_PATH}/lib

# ===============================
# Stage 3: Go build stage
# ===============================
FROM binary-deps AS build

WORKDIR /build

ARG SERVICE_NAME SERVICE_VERSION TARGETOS TARGETARCH

# Build all binaries with optimized cache mounts
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags "-w -X main.serviceVersion=${SERVICE_VERSION} -X main.serviceName=${SERVICE_NAME}" \
    -tags=ocr,onnx -o /${SERVICE_NAME} ./cmd/main && \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags "-w -X main.serviceVersion=${SERVICE_VERSION} -X main.serviceName=${SERVICE_NAME}-worker" \
    -tags=ocr,onnx -o /${SERVICE_NAME}-worker ./cmd/worker && \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags "-w -X main.serviceVersion=${SERVICE_VERSION} -X main.serviceName=${SERVICE_NAME}-migrate" \
    -tags=ocr,onnx -o /${SERVICE_NAME}-migrate ./cmd/migration && \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags "-w -X main.serviceVersion=${SERVICE_VERSION} -X main.serviceName=${SERVICE_NAME}-init" \
    -tags=ocr,onnx -o /${SERVICE_NAME}-init ./cmd/init

# ===============================
# Stage 4: Runtime dependencies
# ===============================
FROM debian:bullseye-slim AS runtime-deps

# Install all runtime dependencies in one layer
RUN apt update && \
    apt install -y \
    build-essential \
    curl \
    wget \
    unzip \
    xz-utils \
    python3 \
    python3-venv \
    python3-dev \
    libgeos++-dev \
    poppler-utils \
    wv \
    unrtf \
    tidy \
    tesseract-ocr \
    libtesseract-dev \
    libreoffice \
    libsoxr-dev \
    chromium \
    qpdf && \
    rm -rf /var/lib/apt/lists/*

# Create Python virtual environment and install packages
RUN python3 -m venv /opt/venv && \
    /opt/venv/bin/pip install --no-cache-dir pdfplumber mistral-common tokenizers docling==2.18.0

# ===============================
# Stage 5: Model artifacts
# ===============================
FROM runtime-deps AS model-artifacts

# Copy FFmpeg and ONNX runtime from build stage
COPY --from=build --chown=nobody:nogroup /usr/local/bin/ffmpeg /usr/local/bin/ffmpeg
COPY --from=build --chown=nobody:nogroup /usr/local/bin/ffprobe /usr/local/bin/ffprobe

ENV ONNXRUNTIME_ROOT_PATH=/usr/local/onnxruntime
COPY --from=build --chown=nobody:nogroup /usr/local/onnxruntime ${ONNXRUNTIME_ROOT_PATH}

# Set environment variables and create symlinks for ONNX Runtime
ENV C_INCLUDE_PATH=${ONNXRUNTIME_ROOT_PATH}/include
RUN ln -s ${ONNXRUNTIME_ROOT_PATH}/lib/libonnxruntime.so* /usr/lib/

# Set up model directories and download artifacts
ENV BASE_DOCLING_PATH=/home/nobody
ENV DOCLING_ARTIFACTS_PATH=${BASE_DOCLING_PATH}/docling-artifacts

RUN mkdir -p ${BASE_DOCLING_PATH}/.EasyOCR/model && \
    chown -R nobody:nogroup ${BASE_DOCLING_PATH}

# Download EasyOCR models and set up Docling artifacts
RUN apt update && \
    apt install -y wget unzip && \
    wget https://github.com/JaidedAI/EasyOCR/releases/download/v1.3/latin_g2.zip && \
    wget https://github.com/JaidedAI/EasyOCR/releases/download/pre-v1.1.6/craft_mlt_25k.zip && \
    unzip latin_g2.zip -d ${BASE_DOCLING_PATH}/.EasyOCR/model/ && \
    unzip craft_mlt_25k.zip -d ${BASE_DOCLING_PATH}/.EasyOCR/model/ && \
    rm latin_g2.zip craft_mlt_25k.zip && \
    # Download Docling artifacts
    echo "from docling.pipeline.standard_pdf_pipeline import StandardPdfPipeline" > import_artifacts.py && \
    echo "StandardPdfPipeline.download_models_hf(local_dir='${DOCLING_ARTIFACTS_PATH}')" >> import_artifacts.py && \
    /opt/venv/bin/python import_artifacts.py && \
    rm import_artifacts.py && \
    # Final cleanup
    apt remove -y wget unzip && \
    apt autoremove -y && \
    rm -rf /var/lib/apt/lists/*

# ===============================
# Stage 6: Final runtime image
# ===============================
FROM model-artifacts AS final

ARG SERVICE_NAME SERVICE_VERSION

USER nobody:nogroup

ENV HOME=${BASE_DOCLING_PATH}
WORKDIR /${SERVICE_NAME}

ENV GODEBUG=tlsrsakex=1

# Copy all built binaries from build stage
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-migrate ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-init ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-worker ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME} ./

# Copy application files
COPY --chown=nobody:nogroup ./config ./config
COPY --chown=nobody:nogroup ./pkg/db/migration ./pkg/db/migration

# Set up ONNX model and environment variable
COPY --chown=nobody:nogroup ./pkg/component/resources/onnx/silero_vad.onnx /${SERVICE_NAME}/pkg/component/resources/onnx/silero_vad.onnx
ENV ONNX_MODEL_FOLDER_PATH=/${SERVICE_NAME}/pkg/component/resources/onnx

ENV SERVICE_NAME=${SERVICE_NAME}
ENV SERVICE_VERSION=${SERVICE_VERSION}
