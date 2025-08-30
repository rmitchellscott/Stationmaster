FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.6.1 AS xx

# Ruby dependencies build
FROM --platform=$BUILDPLATFORM ruby:3.4-alpine AS ruby-builder
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache \
    build-base \
    git

# Copy gem files
COPY Gemfile Gemfile.lock ./

# Install gems with deployment flag for faster installs
RUN bundle config set --local deployment 'true' && \
    bundle config set --local without 'development test' && \
    bundle install --jobs $(nproc) --retry 3

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

# Download TRMNL locale files for i18n compatibility
RUN apk add --no-cache git && \
    echo "Cloning TRMNL i18n repository..." && \
    git clone --depth 1 --filter=blob:none --sparse https://github.com/usetrmnl/trmnl-i18n.git /tmp/trmnl-i18n && \
    cd /tmp/trmnl-i18n && \
    git sparse-checkout set lib/trmnl/i18n/locales/custom_plugins && \
    mkdir -p /app/internal/locales && \
    cp lib/trmnl/i18n/locales/custom_plugins/*.yml /app/internal/locales/ && \
    echo "Successfully copied $(ls /app/internal/locales/*.yml | wc -l) locale files" && \
    rm -rf /tmp/trmnl-i18n && \
    apk del git

# Build args for version injection
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
ARG TARGETPLATFORM

RUN --mount=type=cache,target=/root/.cache \
    CGO_ENABLED=0 xx-go build \
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
      ruby \
      ruby-bundler \
    && update-ca-certificates

WORKDIR /app

# Copy pre-built binaries and assets
COPY --from=stationmaster-builder /app/stationmaster .
COPY --from=stationmaster-builder /app/images ./images

# Copy pre-built Ruby gems and configuration
COPY --from=ruby-builder /app/vendor ./vendor
COPY Gemfile Gemfile.lock ./

# Set bundle config to use pre-built gems
RUN bundle config set --local deployment 'true' && \
    bundle config set --local without 'development test' && \
    bundle config set --local path 'vendor/bundle' && \
    bundle check || bundle install --local

# Copy Ruby scripts
COPY scripts/ ./scripts/
RUN chmod +x ./scripts/*.rb

# Copy TRMNL plugins
COPY trmnl-plugins/ ./trmnl-plugins/

# Create data directory and setup for Chromium
RUN mkdir -p /data /app/static/rendered \
    && addgroup -g 1000 -S appuser \
    && adduser -u 1000 -S appuser -G appuser \
    && chown -R appuser:appuser /app /data


USER appuser

EXPOSE 8000
CMD ["./stationmaster"]
