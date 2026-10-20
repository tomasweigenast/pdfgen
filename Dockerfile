# syntax=docker/dockerfile:1

# Compile on the runner architecture; Go cross-compiles for TARGETOS/TARGETARCH.
FROM --platform=$BUILDPLATFORM golang:1.27.1-bookworm AS build

WORKDIR /src

# Keep module downloads cached when only application code changes.
COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH

# Release builds pass the same version and commit used for the archives.
ARG VERSION=dev
ARG COMMIT=unknown

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    GOAMD64=v1 \
    go build \
        -trimpath \
        -ldflags="-s -w -buildid= \
            -X github.com/tomasweigenast/pdfgen/internal/pdf/metadata.Version=$VERSION \
            -X main.commit=$COMMIT" \
        -o /out/pdfgen \
        ./cmd/pdfgen

# Fonts and CA certificates are data files, so this stage can also run natively.
FROM --platform=$BUILDPLATFORM debian:13-slim AS assets

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        fonts-dejavu-core \
    && rm -rf /var/lib/apt/lists/*

# The runtime only needs the static binary, fonts and certificates.
FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=build /out/pdfgen /usr/local/bin/pdfgen

COPY --from=assets \
    /etc/ssl/certs/ca-certificates.crt \
    /etc/ssl/certs/ca-certificates.crt

COPY --from=assets \
    /usr/share/fonts/truetype/dejavu \
    /usr/share/fonts/truetype/dejavu

# Retain the font license and the notices for bundled dictionaries.
COPY --from=assets \
    /usr/share/doc/fonts-dejavu-core \
    /usr/share/doc/third-party/fonts-dejavu-core

COPY LICENSE NOTICE /usr/share/doc/pdfgen/

COPY internal/text/hyphen/dictionaries/ /usr/share/doc/third-party/hyphenation/

# Font discovery writes a cache; the non-root user can write under /tmp.
ENV XDG_CACHE_HOME=/tmp/pdfgen-cache

USER 65532:65532
WORKDIR /work

ENTRYPOINT ["/usr/local/bin/pdfgen"]
CMD ["--help"]
