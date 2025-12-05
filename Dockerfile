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


# Ruby setup stage - use Alpine's Ruby and install gems
FROM alpine:3.22 AS ruby-setup

# Install Ruby and build dependencies for gems
RUN apk add --no-cache \
    ruby \
    ruby-dev \
    build-base

# Install required Ruby gems
RUN gem install \
    liquid \
    trmnl-liquid \
    trmnl-i18n \
    --no-document

# Final image
FROM alpine:3.22

ARG S6_OVERLAY_VERSION=3.2.1.0
ARG TARGETARCH

# Install minimal runtime dependencies including Ruby
RUN apk add --no-cache \
      ca-certificates \
      postgresql-client \
      tzdata \
      ruby \
      curl \
      xz \
    && update-ca-certificates \
    && case ${TARGETARCH} in \
         "amd64")  S6_ARCH=x86_64  ;; \
         "arm64")  S6_ARCH=aarch64 ;; \
         *)        S6_ARCH=x86_64  ;; \
       esac \
    && curl -sSL https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-noarch.tar.xz | tar -C / -Jxpf - \
    && curl -sSL https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-${S6_ARCH}.tar.xz | tar -C / -Jxpf - \
    && apk del curl xz

WORKDIR /app

# Copy pre-built Go binary and assets
COPY --from=stationmaster-builder /app/stationmaster .
COPY --from=stationmaster-builder /app/images ./images

# Copy installed gems from ruby-setup stage
COPY --from=ruby-setup /usr/lib/ruby/gems /usr/lib/ruby/gems

# Copy Ruby scripts
COPY embedded_ruby/scripts/ ./scripts/
RUN chmod +x ./scripts/start.sh ./scripts/liquid_server.rb

# Copy s6-overlay service definitions
COPY embedded_ruby/s6-rc.d/ /etc/s6-overlay/s6-rc.d/
RUN chmod +x /etc/s6-overlay/s6-rc.d/liquid-renderer/run \
             /etc/s6-overlay/s6-rc.d/stationmaster/run

EXPOSE 8000
ENTRYPOINT ["/init"]
