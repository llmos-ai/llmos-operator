VERSION --arg-scope-and-set 0.8

LET go_version = 1.23
LET distro = alpine3.20

FROM golang:${go_version}-${distro}
ARG --global ALPINE=3.20
ARG --global ALPINE_DIND=earthly/dind:alpine-3.20
ARG --global REGISTRY=
ARG --global DOCKER_REGISTRY=
ARG --global TAG=
ARG --global VERSION=
ARG --global CHART_VERSION=
ARG --global HELM_VERSION=v3.15.3
ARG --global AWS_ACCESS_KEY_ID=
ARG --global AWS_SECRET_ACCESS_KEY=
ARG --global AWS_DEFAULT_REGION=
ARG --global S3_BUCKET_NAME=
ARG --global UPLOAD_CHARTS=
ARG --global K3S_TAG=v1.31.0+k3s1

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

package-all-upgrade-image:
    BUILD --pass-args \
        --platform=linux/amd64 \
        --platform=linux/arm64 \
        +package-upgrade

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
    COPY package/installer-run.sh dist/run.sh
    RUN sed -i "s/\${CHART_VERSION}/$CHART_VERSION/g" dist/run.sh
    RUN cat dist/run.sh
    SAVE ARTIFACT dist AS LOCAL dist/llmos-charts

package-installer:
    FROM scratch
    COPY +build-installer/dist/helm /
    COPY +build-installer/dist/llmos-charts/*.tgz /
    COPY +build-installer/dist/run.sh /run.sh
    SAVE IMAGE --cache-from ${DOCKER_REGISTRY}/system-installer-llmos-operator:${TAG} --push ${DOCKER_REGISTRY}/system-installer-llmos-operator:${TAG}
    SAVE IMAGE --cache-from ${REGISTRY}/system-installer-llmos-operator:${TAG} --push ${REGISTRY}/system-installer-llmos-operator:${TAG}

build-system-charts:
    FROM nginx:alpine$ALPINE
    WORKDIR llmos-repo
    RUN apk update && apk add --no-cache git helm yq jq bash
    COPY . .
    RUN ./scripts/build-charts-repo
    RUN ls -la dist/system-charts-repo
    RUN [ -e "dist/system-charts-repo/index.yaml" ] && echo "found index.yaml" || exit 1
    SAVE ARTIFACT dist/system-charts-repo AS LOCAL dist/system-charts-repo

package-system-charts-repo:
    FROM nginx:alpine$ALPINE
    WORKDIR /usr/share/nginx/html
    COPY +build-system-charts/system-charts-repo .
    RUN [ -e "/usr/share/nginx/html/index.yaml" ] && echo "found index.yaml" || exit 1
    EXPOSE 80
    CMD ["nginx", "-g", "daemon off;"]
    SAVE IMAGE --cache-from ${REGISTRY}/system-charts-repo:${TAG} --push ${REGISTRY}/system-charts-repo:${TAG}
    SAVE IMAGE --cache-from ${DOCKER_REGISTRY}/system-charts-repo:${TAG} --push ${DOCKER_REGISTRY}/system-charts-repo:${TAG}
    IF [ "$VERSION" != "$TAG" ]
    SAVE IMAGE --cache-from ${REGISTRY}/system-charts-repo:${VERSION} --push ${REGISTRY}/system-charts-repo:${VERSION}
    SAVE IMAGE --cache-from ${DOCKER_REGISTRY}/system-charts-repo:${VERSION} --push ${DOCKER_REGISTRY}/system-charts-repo:${VERSION}
    END

build-upgrade:
    ARG TARGETARCH # system arg
    FROM alpine:$ALPINE
    WORKDIR /verify
    RUN set -x \
     && apk upgrade -U \
     && apk add \
        curl file \
     && apk cache clean \
     && rm -rf /var/cache/apk/*
    RUN curl -O -sfL https://github.com/k3s-io/k3s/releases/download/${K3S_TAG}/sha256sum-${TARGETARCH}.txt
    RUN if [ "${TARGETARCH}" == "amd64" ]; then \
          export ARTIFACT="k3s"; \
        elif [ "${TARGETARCH}" == "arm" ]; then \
          export ARTIFACT="k3s-armhf"; \
        elif [ "${TARGETARCH}" == "arm64" ]; then \
          export ARTIFACT="k3s-arm64"; \
        elif [ "${TARGETARCH}" == "s390x" ]; then \
          export ARTIFACT="k3s-s390x"; \
        fi \
     && curl --output ${ARTIFACT}  --fail --location https://github.com/k3s-io/k3s/releases/download/${K3S_TAG}/${ARTIFACT} \
     && grep -E " k3s(-arm\w*|-s390x)?$" sha256sum-${TARGETARCH}.txt | sha256sum -c \
     && mv -vf ${ARTIFACT} /opt/k3s \
     && chmod +x /opt/k3s \
     && file /opt/k3s
    SAVE ARTIFACT /opt AS LOCAL dist/k3s

package-upgrade:
    FROM alpine:$ALPINE
    ARG K3S_TAG
    ENV LLMOS_SERVER_VERSION ${VERSION}
    RUN apk upgrade -U \
     && apk add \
        jq libselinux-utils procps \
     && apk cache clean \
     && rm -rf /var/cache/apk/*
    COPY +build-upgrade/opt/k3s /opt/k3s
    COPY package/upgrade-node.sh /bin/upgrade-node.sh
    ENTRYPOINT ["/bin/upgrade-node.sh"]
    SAVE IMAGE --cache-from ${REGISTRY}/node-upgrade:${TAG} --push ${REGISTRY}/node-upgrade:${TAG}
    SAVE IMAGE --cache-from ${DOCKER_REGISTRY}/node-upgrade:${TAG} --push ${DOCKER_REGISTRY}/node-upgrade:${TAG}
    IF [ "$VERSION" != "$TAG" ]
    SAVE IMAGE --cache-from ${REGISTRY}/node-upgrade:${VERSION} --push ${REGISTRY}/node-upgrade:${VERSION}
    SAVE IMAGE --cache-from ${DOCKER_REGISTRY}/node-upgrade:${VERSION} --push ${DOCKER_REGISTRY}/node-upgrade:${VERSION}
    END
