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
ARG --global AWS_ACCESS_KEY_ID=
ARG --global AWS_SECRET_ACCESS_KEY=
ARG --global AWS_DEFAULT_REGION=
ARG --global S3_BUCKET_NAME=
ARG --global UPLOAD_CHARTS=

WORKDIR /llmos-operator

package-all-installer:
    BUILD --pass-args \
        --platform=linux/amd64 \
        --platform=linux/arm64 \
        +package-installer

package-all-system-charts-repo:
    BUILD --pass-args \
        --platform=linux/amd64 \
        --platform=linux/arm64 \
        +package-system-charts-repo

build-installer:
    ARG TARGETARCH # system arg
    FROM alpine:$ALPINE
    WORKDIR llmos-operator
    ARG REGISTRY
    RUN apk update && apk add --no-cache git yq jq bash curl aws-cli
    ENV HELM_URL=https://get.helm.sh/helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz
    # set up helm 3
    RUN curl -sfL ${HELM_URL} | tar xvzf - --strip-components=1 -C /usr/bin
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

build-system-charts:
    FROM nginx:alpine$ALPINE
    WORKDIR llmos-repo
    RUN apk update && apk add --no-cache git helm yq jq bash
    COPY . .
    RUN ./scripts/chart/system-charts
    RUN ls -la dist/system-charts
    RUN [ -e "dist/system-charts/index.yaml" ] && echo "found index.yaml" || exit 1
    SAVE ARTIFACT dist/system-charts AS LOCAL dist/system-charts

package-system-charts-repo:
    FROM nginx:alpine$ALPINE
    WORKDIR /usr/share/nginx/html
    COPY +build-system-charts/system-charts .
    RUN [ -e "/usr/share/nginx/html/index.yaml" ] && echo "found index.yaml" || exit 1
    EXPOSE 80
    CMD ["nginx", "-g", "daemon off;"]
    SAVE IMAGE --cache-from ${REGISTRY}/system-charts-repo:${TAG} --push ${REGISTRY}/system-charts-repo:${TAG}
    IF [ "$VERSION" != "$TAG" ]
    SAVE IMAGE --cache-from ${REGISTRY}/system-charts-repo:${VERSION} --push ${REGISTRY}/system-charts-repo:${VERSION}
    END
