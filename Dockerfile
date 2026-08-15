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
ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/root/.cache \
    CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build \
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

# Install required Ruby gems.
# actionview + activesupport render the external plugins' ERB templates. They are
# deliberately installed without rails: ActionView resolves the templates, their
# partial chains and their locals standalone, which is what lets those templates
# move across unchanged. Adding rails would pull a web stack this process never serves.
RUN gem install \
    liquid \
    trmnl-liquid \
    trmnl-i18n \
    actionview \
    activesupport \
    httparty \
    icalendar \
    icalendar-recurrence \
    google-apis-calendar_v3 \
    google-apis-analyticsdata_v1beta \
    google-apis-youtube_analytics_v2 \
    --no-document

# SEC ticker data for stock_price. Fetched in its own stage because the final image
# deletes curl after unpacking s6.
FROM alpine:3.22 AS ticker-data
RUN apk add --no-cache curl ruby
WORKDIR /work
COPY embedded_ruby/scripts/download-ticker-data ./
RUN chmod +x download-ticker-data && ./download-ticker-data

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

# The external plugins and the locale keys their templates ask for.
COPY embedded_ruby/plugins/ ./plugins/
COPY embedded_ruby/locales/ ./locales/

# stock_price reads db/data/ticker-name.json by a path relative to the working
# directory, so the file has to sit under /app and the s6 run script has to cd here.
COPY --from=ticker-data /work/db ./db

# Copy s6-overlay service definitions
COPY embedded_ruby/s6-rc.d/ /etc/s6-overlay/s6-rc.d/
RUN chmod +x /etc/s6-overlay/s6-rc.d/liquid-renderer/run \
             /etc/s6-overlay/s6-rc.d/stationmaster/run

EXPOSE 8000
ENTRYPOINT ["/init"]
