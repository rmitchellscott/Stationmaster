FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.6.1 AS xx

# Frontend build
FROM --platform=$BUILDPLATFORM node:24-alpine AS ui-builder
WORKDIR /app

COPY ui/package.json ui/package-lock.json ui/
RUN cd ui && npm ci
COPY ui/ ui/
COPY locales/ locales/
RUN cd ui && npm run build

FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS go-base
WORKDIR /app
COPY --from=xx / /

# Build backend
FROM --platform=$BUILDPLATFORM go-base AS stationmaster-builder

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=ui-builder /app/ui/dist ./ui/dist

# Download TRMNL assets at build time
RUN apk add --no-cache curl bash \
    && chmod +x ./scripts/download-trmnl-assets.sh \
    && ./scripts/download-trmnl-assets.sh \
    && apk del curl bash

# Build args for version injection
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
ARG TARGETPLATFORM

RUN --mount=type=cache,target=/root/.cache \
    CGO_ENABLED=0 xx-go build \
    -tags production \
    -ldflags="-w -s \
        -X github.com/rmitchellscott/stationmaster/internal/version.Version=${VERSION} \
        -X github.com/rmitchellscott/stationmaster/internal/version.GitCommit=${GIT_COMMIT} \
        -X github.com/rmitchellscott/stationmaster/internal/version.BuildDate=${BUILD_DATE}" \
    -trimpath


# Final image
FROM alpine:3.22

# Install minimal runtime dependencies
RUN apk add --no-cache \
      ca-certificates \
      postgresql-client \
      tzdata \
    && update-ca-certificates

WORKDIR /app

# Copy pre-built binaries and assets
COPY --from=stationmaster-builder /app/stationmaster .
COPY --from=stationmaster-builder /app/images ./images

EXPOSE 8000
CMD ["./stationmaster"]
