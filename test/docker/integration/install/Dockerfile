FROM ubuntu:22.04 as install
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

WORKDIR /agent
COPY ./ /agent
COPY ./test/testdata/agentless-entrypoint.sh /agent/agentless-entrypoint.sh

RUN set -x \
    && addgroup --system --gid 101 nginx \
    && adduser --system --disabled-login --ingroup nginx --no-create-home --home /nonexistent --gecos "nginx user" --shell /bin/false --uid 101 nginx \
    && apt-get update \
    && apt-get install --no-install-recommends --no-install-suggests -y \
                        ca-certificates \
                        gnupg1 \
                        lsb-release \
                        git \
                        wget \
                        make \
                        curl \
                        vim \
                        golang \
                        nginx

RUN chmod +x /agent/agentless-entrypoint.sh
STOPSIGNAL SIGTERM

ENTRYPOINT ["/agent/agentless-entrypoint.sh"]
