FROM opensuse/leap:15.6

WORKDIR /var/lib/llmos-ai/llmos-controller

RUN zypper ar -G https://download.opensuse.org/repositories/utilities/15.6/utilities.repo || true && \
    zypper ref

RUN ARCH=$(uname -m); \
    [[ "${ARCH}" == "aarch64" ]] && ARCH="arm64"; \
    zypper --non-interactive install --no-recommends -- git unzip tar gzip && \
    zypper clean --all && \
    mkdir -p /var/lib/llmos-ai/llmos-controller

# Add dumb-init
ENV INIT_VERSION 1.2.5
RUN ARCH=$(uname -m); \
    curl -sfL https://github.com/Yelp/dumb-init/releases/download/v${INIT_VERSION}/dumb-init_${INIT_VERSION}_${ARCH} -o dumb-init && \
    chmod +x dumb-init && \
    mv dumb-init /usr/bin/dumb-init

ENV UI_VERSION latest
# also update the api-ui-version in pkg/settings/settings.go when updating the version here.
ENV API_UI_VERSION 1.1.10

RUN mkdir -p /usr/share/llmos-ai/llmos-controller && \
    cd /usr/share/llmos-ai/llmos-controller && \
    curl -sL https://releases.1block.ai/llmos-ui/${UI_VERSION}.tar.gz | tar xvzf - --strip-components=2 && \
    mkdir -p /usr/share/llmos-ai/llmos-controller/api-ui && \
    cd /usr/share/llmos-ai/llmos-controller/api-ui && \
    curl -sL https://releases.rancher.com/api-ui/${API_UI_VERSION}.tar.gz | tar xvzf - --strip-components=1 && \
    cd /var/lib/llmos-ai/llmos-controller

COPY package/entrypoint.sh llmos-controller /usr/bin/
RUN chmod +x /usr/bin/entrypoint.sh

VOLUME /var/lib/llmos-ai/llmos-controller
ENTRYPOINT ["entrypoint.sh"]
