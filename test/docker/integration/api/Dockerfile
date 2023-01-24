FROM ubuntu:22.04 as install
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

ARG PACKAGE

WORKDIR /agent
COPY ./ /agent
COPY ./test/testdata/entrypoint.sh /agent/entrypoint.sh
COPY ./test/testdata/configs/integration/api/nginx-agent.conf /agent/nginx-agent.conf
COPY ./test/testdata/configs/integration/api/nginx.conf /agent/nginx.conf

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
                        nginx \
    && apt-get update \
    && apt install -y -f /agent/build/$PACKAGE.deb

# run the nginx and agent
FROM install as runtime

COPY --from=install /agent/entrypoint.sh /agent/entrypoint.sh
COPY --from=install /agent/nginx-agent.conf /etc/nginx-agent/nginx-agent.conf

RUN chmod +x /agent/entrypoint.sh
STOPSIGNAL SIGTERM
EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]
