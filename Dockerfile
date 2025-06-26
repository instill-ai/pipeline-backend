ARG GOLANG_VERSION=1.24.2
FROM golang:${GOLANG_VERSION}-bullseye AS build

ARG TARGETOS TARGETARCH

RUN apt-get update && apt-get install -y \
    build-essential \
    libleptonica-dev \
    libtesseract-dev \
    libsoxr-dev \
    && rm -rf /var/lib/apt/lists/*

# Install FFmpeg Static Build
RUN FFMPEG_ARCH=$([ "$TARGETARCH" = "arm64" ] && echo "arm64" || echo "amd64") && \
    wget https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-${FFMPEG_ARCH}-static.tar.xz && \
    tar xvf ffmpeg-release-${FFMPEG_ARCH}-static.tar.xz && \
    mv ffmpeg-*-static/ffmpeg /usr/local/bin/ && \
    mv ffmpeg-*-static/ffprobe /usr/local/bin/ && \
    rm -rf ffmpeg-*

# Install ONNX Runtime (latest release)
ENV ONNXRUNTIME_ROOT_PATH=/usr/local/onnxruntime
RUN apt update && \
    apt install -y wget jq && \
    VERSION=v1.20.1 && \
    ONNX_ARCH=$([ "$TARGETARCH" = "arm64" ] && echo "aarch64" || echo "x64") && \
    wget https://github.com/microsoft/onnxruntime/releases/download/${VERSION}/onnxruntime-linux-${ONNX_ARCH}-${VERSION#v}.tgz && \
    tar -xzf onnxruntime-linux-${ONNX_ARCH}-${VERSION#v}.tgz && \
    mv onnxruntime-linux-${ONNX_ARCH}-${VERSION#v} ${ONNXRUNTIME_ROOT_PATH} && \
    rm onnxruntime-linux-${ONNX_ARCH}-${VERSION#v}.tgz && \
    apt remove -y wget jq && \
    apt autoremove -y && \
    rm -rf /var/lib/apt/lists/*

# Set environment variables and create symlinks for ONNX Runtime
ENV C_INCLUDE_PATH=${ONNXRUNTIME_ROOT_PATH}/include
ENV LD_RUN_PATH=${ONNXRUNTIME_ROOT_PATH}/lib
ENV LIBRARY_PATH=${ONNXRUNTIME_ROOT_PATH}/lib

WORKDIR /build

ARG SERVICE_NAME SERVICE_VERSION TARGETOS TARGETARCH

RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags "-w -X main.version=${SERVICE_VERSION} -X main.serviceName=${SERVICE_NAME}" \
    -tags=ocr,onnx -o /${SERVICE_NAME} ./cmd/main

RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \    
    go build -ldflags "-w -X main.serviceVersion=${SERVICE_VERSION} -X main.serviceName=${SERVICE_NAME}-worker" \
    -tags=ocr,onnx -o /${SERVICE_NAME}-worker ./cmd/worker

RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags "-w -X main.version=${SERVICE_VERSION} -X main.serviceName=${SERVICE_NAME}-migrate" \
    -tags=ocr,onnx -o /${SERVICE_NAME}-migrate ./cmd/migration

RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags "-w -X main.version=${SERVICE_VERSION} -X main.serviceName=${SERVICE_NAME}-init" \
    -tags=ocr,onnx -o /${SERVICE_NAME}-init ./cmd/init

FROM debian:bullseye-slim

# Install Python, create virtual environment, install pdfplumber and Docling
RUN apt update && \
    apt install -y \
    build-essential \
    curl \
    wget \
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
    python3 -m venv /opt/venv && \
    /opt/venv/bin/pip install pdfplumber mistral-common tokenizers docling==2.18.0 && \
    rm -rf /var/lib/apt/lists/*

# Copy FFmpeg from build stage
COPY --from=build --chown=nobody:nogroup /usr/local/bin/ffmpeg /usr/local/bin/ffmpeg
COPY --from=build --chown=nobody:nogroup /usr/local/bin/ffprobe /usr/local/bin/ffprobe

# Copy ONNX runtime from build stage
ENV ONNXRUNTIME_ROOT_PATH=/usr/local/onnxruntime
COPY --from=build --chown=nobody:nogroup /usr/local/onnxruntime ${ONNXRUNTIME_ROOT_PATH}

# Set environment variables and create symlinks for ONNX Runtime
ENV C_INCLUDE_PATH=${ONNXRUNTIME_ROOT_PATH}/include
RUN ln -s ${ONNXRUNTIME_ROOT_PATH}/lib/libonnxruntime.so* /usr/lib/

# Docling will need a $HOME-prefixed path to write cache files. We'll also put
# the prefetched model artifacts there.
ENV BASE_DOCLING_PATH=/home/nobody
RUN mkdir -p ${BASE_DOCLING_PATH}/.EasyOCR/model && chown -R nobody:nogroup ${BASE_DOCLING_PATH}

RUN apt update && \
    apt install -y unzip && \
    wget https://github.com/JaidedAI/EasyOCR/releases/download/v1.3/latin_g2.zip && \
    unzip latin_g2.zip -d ${BASE_DOCLING_PATH}/.EasyOCR/model/ && \
    rm latin_g2.zip && \
    wget https://github.com/JaidedAI/EasyOCR/releases/download/pre-v1.1.6/craft_mlt_25k.zip && \
    unzip craft_mlt_25k.zip -d ${BASE_DOCLING_PATH}/.EasyOCR/model/ && \
    rm craft_mlt_25k.zip && \
    apt remove -y unzip && \
    apt autoremove -y && \
    rm -rf /var/lib/apt/lists/*

ENV DOCLING_ARTIFACTS_PATH=${BASE_DOCLING_PATH}/docling-artifacts
RUN echo "from docling.pipeline.standard_pdf_pipeline import StandardPdfPipeline" > import_artifacts.py
RUN echo "StandardPdfPipeline.download_models_hf(local_dir='${DOCLING_ARTIFACTS_PATH}')" >> import_artifacts.py
RUN /opt/venv/bin/python import_artifacts.py && rm import_artifacts.py

USER nobody:nogroup

ARG SERVICE_NAME SERVICE_VERSION

ENV HOME=${BASE_DOCLING_PATH}
WORKDIR /${SERVICE_NAME}

ENV GODEBUG=tlsrsakex=1

COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-migrate ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-init ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-worker ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME} ./

COPY --chown=nobody:nogroup ./config ./config
COPY --chown=nobody:nogroup ./pkg/db/migration ./pkg/db/migration

# Set up ONNX model and environment variable
COPY --chown=nobody:nogroup ./pkg/component/resources/onnx/silero_vad.onnx /${SERVICE_NAME}/pkg/component/resources/onnx/silero_vad.onnx
ENV ONNX_MODEL_FOLDER_PATH=/${SERVICE_NAME}/pkg/component/resources/onnx

ENV SERVICE_NAME=${SERVICE_NAME}
ENV SERVICE_VERSION=${SERVICE_VERSION}
