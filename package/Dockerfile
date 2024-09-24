FROM registry.opensuse.org/opensuse/leap:15.6

WORKDIR /var/lib/llmos/llmos-operator

RUN zypper ar -G https://download.opensuse.org/repositories/utilities/15.6/utilities.repo || true && \
    zypper ref

RUN zypper -n in git unzip tar gzip && \
    zypper -n clean -a && rm -rf /tmp/* /var/tmp/* /usr/share/doc/packages/* && \
    mkdir -p /var/lib/llmos/llmos-operator

# Add dumb-init
ENV INIT_VERSION 1.2.5
RUN ARCH=$(uname -m); \
    curl -sfL https://github.com/Yelp/dumb-init/releases/download/v${INIT_VERSION}/dumb-init_${INIT_VERSION}_${ARCH} -o dumb-init && \
    chmod +x dumb-init && \
    mv dumb-init /usr/bin/dumb-init

ENV LLMOS_UI_VERSION latest
# also update the api-ui-version in pkg/settings/settings.go when updating the version here.
ENV LLMOS_API_UI_VERSION 1.1.11

ARG VERSION
ARG DATE
ENV LLMOS_SERVER_VERSION ${VERSION}
ENV LLMOS_BUILD_DATE ${DATE}
RUN mkdir -p /usr/share/llmos/llmos-operator && \
    cd /usr/share/llmos/llmos-operator && \
    curl -sL https://releases.1block.ai/dashboard/${LLMOS_UI_VERSION}.tar.gz | tar xvzf - --strip-components=2 && \
    mkdir -p /usr/share/llmos/llmos-operator/api-ui && \
    cd /usr/share/llmos/llmos-operator/api-ui && \
    curl -sL https://releases.1block.ai/api-ui/${LLMOS_API_UI_VERSION}.tar.gz | tar xvzf - --strip-components=1 && \
    cd /var/lib/llmos/llmos-operator

COPY package/entrypoint.sh llmos-operator /usr/bin/
RUN chmod +x /usr/bin/entrypoint.sh

VOLUME /var/lib/llmos/llmos-operator
ENTRYPOINT ["entrypoint.sh"]
