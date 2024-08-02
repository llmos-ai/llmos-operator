VERSION --arg-scope-and-set 0.8

LET go_version = 1.22
LET distro = alpine3.19

FROM golang:${go_version}-${distro}
ARG --global ALPINE=3.19
ARG --global ALPINE_DIND=earthly/dind:alpine-3.19
ARG --global REGISTRY=
ARG --global DOCKER_REGISTRY=
ARG --global TAG=
ARG --global VERSION=
ARG --global HELM_VERSION=v3.15.3

WORKDIR /llmos-operator

package-all-installer:
    BUILD --pass-args \
        --platform=linux/amd64 \
        --platform=linux/arm64 \
        +package-installer

build-all-installer:
    BUILD \
        --platform=linux/amd64 \
        --platform=linux/arm64 \
        +build-installer

build-installer:
    ARG TARGETARCH # system arg
    FROM alpine:$ALPINE
    WORKDIR llmos-operator
    ARG REGISTRY
    RUN apk update && apk add --no-cache git yq jq bash curl
    ENV HELM_URL=https://get.helm.sh/helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz
    # set up helm 3
    RUN curl ${HELM_URL} | tar xvzf - --strip-components=1 -C /usr/bin
    COPY . .
    RUN ./scripts/ci
    RUN cp /usr/bin/helm dist/helm
    SAVE ARTIFACT dist AS LOCAL dist/charts

package-installer:
    FROM scratch
    COPY +build-installer/dist/helm /
    COPY +build-installer/dist/charts/*.tgz /
    COPY package/installer-run.sh /run.sh
    SAVE IMAGE --cache-from ${DOCKER_REGISTRY}/system-installer-llmos-operator:${TAG} --push ${DOCKER_REGISTRY}/system-installer-llmos-operator:${TAG}
    SAVE IMAGE --cache-from ${DOCKER_REGISTRY}/system-installer-llmos-operator:${VERSION} --push ${DOCKER_REGISTRY}/system-installer-llmos-operator:${VERSION}
