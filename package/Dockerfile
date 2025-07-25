# Multi-stage build for optimized container image
FROM golang:1.24-alpine AS ui-downloader

# Download UI assets in separate stage for better caching
ARG LLMOS_UI_VERSION
ENV LLMOS_UI_VERSION ${LLMOS_UI_VERSION:-latest}
ENV LLMOS_API_UI_VERSION 1.1.11

RUN apk add --no-cache curl tar gzip && \
    mkdir -p /tmp/ui/dashboard /tmp/ui/api-ui && \
    cd /tmp/ui/dashboard && \
    curl -sL https://releases.1block.ai/dashboard/${LLMOS_UI_VERSION}.tar.gz | tar xzf - --strip-components=2 && \
    cd /tmp/ui/api-ui && \
    curl -sL https://releases.1block.ai/api-ui/${LLMOS_API_UI_VERSION}.tar.gz | tar xzf - --strip-components=1

# Runtime stage with Chainguard Wolfi base image
FROM cgr.dev/chainguard/wolfi-base

# Install basic tools
RUN apk add --no-cache bash tini curl kubectl jq

WORKDIR /var/lib/llmos/llmos-operator

# Copy UI assets from downloader stage
COPY --from=ui-downloader /tmp/ui/dashboard /usr/share/llmos/llmos-operator/
COPY --from=ui-downloader /tmp/ui/api-ui /usr/share/llmos/llmos-operator/api-ui/

# Copy binary and unified entrypoint
COPY llmos-operator /usr/bin/llmos-operator
COPY package/entrypoint.sh /usr/bin/entrypoint.sh

# Set build metadata
ARG VERSION
ARG DATE
ENV LLMOS_SERVER_VERSION=${VERSION}
ENV LLMOS_BUILD_DATE=${DATE}

VOLUME /var/lib/llmos/llmos-operator
ENV LLMOS_MODE=apiserver
ENTRYPOINT ["/usr/bin/entrypoint.sh"]
