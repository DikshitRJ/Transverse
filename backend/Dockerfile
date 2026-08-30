# syntax=docker/dockerfile:1
FROM golang:1.25-bookworm AS builder
WORKDIR /app

# Install ONNX Runtime shared libraries and C headers
RUN wget -q https://github.com/microsoft/onnxruntime/releases/download/v1.18.1/onnxruntime-linux-x64-1.18.1.tgz \
    && tar -xzf onnxruntime-linux-x64-*.tgz \
    && cp -r onnxruntime-linux-x64-*/include/* /usr/local/include/ \
    && cp onnxruntime-linux-x64-*/lib/libonnxruntime.so* /usr/local/lib/ \
    && ldconfig \
    && rm -rf onnxruntime-linux-x64-*

COPY go.mod go.sum* ./
RUN go mod download || true

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -o transverse \
    ./cmd/server

RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -o seed_pipeline \
    ./pipeline/seed

# Final minimal production image
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/local/lib/libonnxruntime.so* /usr/local/lib/
COPY --from=builder /app/transverse /app/transverse
COPY --from=builder /app/seed_pipeline /app/seed_pipeline
COPY --from=builder /app/ingest_pipeline /app/ingest_pipeline

RUN ldconfig

# Data and model volumes (mounted at runtime)
VOLUME ["/app/data", "/app/models"]

EXPOSE 8080
CMD ["/app/transverse"]
