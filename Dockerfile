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

# Install runtime dependencies including Ruby
RUN apk add --no-cache \
      ca-certificates \
      postgresql-client \
      tzdata \
      ruby \
      ruby-dev \
      ruby-bundler \
      build-base \
      git \
    && update-ca-certificates

WORKDIR /app
COPY --from=stationmaster-builder /app/stationmaster .
COPY --from=stationmaster-builder /app/images ./images

# Install Ruby dependencies
COPY Gemfile ./
RUN bundle config set --local without 'development test' && \
    bundle install --jobs 4 --retry 3

# Copy Ruby scripts
COPY scripts/ ./scripts/
RUN chmod +x ./scripts/*.rb

# Create data directory and setup for Chromium
RUN mkdir -p /data /app/static/rendered \
    && addgroup -g 1000 -S appuser \
    && adduser -u 1000 -S appuser -G appuser \
    && chown -R appuser:appuser /app /data


USER appuser

EXPOSE 8000
CMD ["./stationmaster"]
