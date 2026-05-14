# syntax=docker/dockerfile:1.7
# ════════════════════════════════════════════════════════════════════
#  NIGHTCRAWLER v7.0 — Container image
#  Multi-stage build: golang:1.22-alpine → distroless static.
#  Final image targets <25 MB and runs as non-root by default.
# ════════════════════════════════════════════════════════════════════

# ───── Stage 1: builder ─────
FROM golang:1.22-alpine AS builder

WORKDIR /src
ENV CGO_ENABLED=0 GOOS=linux

# Dependency download is its own layer so source-only changes don't
# bust the module cache.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags " \
        -s -w \
        -X github.com/1607-NetEnginee/NightCrawler/internal/version.Version=${VERSION} \
        -X github.com/1607-NetEnginee/NightCrawler/internal/version.Commit=${COMMIT} \
        -X github.com/1607-NetEnginee/NightCrawler/internal/version.BuildDate=${BUILD_DATE}" \
    -o /out/nightcrawler ./cmd/nightcrawler

# ───── Stage 2: distroless runtime ─────
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="nightcrawler"
LABEL org.opencontainers.image.description="Offensive Security Framework — by 1607-NetEnginee / Cyberoutcast"
LABEL org.opencontainers.image.source="https://github.com/1607-NetEnginee/NightCrawler"
LABEL org.opencontainers.image.documentation="https://github.com/1607-NetEnginee/NightCrawler/blob/main/README.md"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.vendor="Cyberoutcast"

COPY --from=builder /out/nightcrawler /usr/local/bin/nightcrawler
COPY --from=builder /src/signatures /etc/nightcrawler/signatures
COPY --from=builder /src/configs/nightcrawler.example.yaml /etc/nightcrawler/config.yaml

USER nonroot:nonroot
WORKDIR /work

ENTRYPOINT ["/usr/local/bin/nightcrawler"]
CMD ["--help"]
