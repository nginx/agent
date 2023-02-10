ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install-nginx
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

ARG AGENT_CONF
ARG NGINX_CONF
ARG ENTRY_POINT

WORKDIR /agent
COPY ./ /agent
COPY $AGENT_CONF /agent/nginx-agent.conf
COPY $NGINX_CONF /agent/nginx.conf
COPY $ENTRY_POINT /agent/entrypoint.sh

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
                        nginx

FROM install-nginx as install-agent

ARG PACKAGE_NAME

COPY --from=install-nginx /agent/nginx.conf /etc/nginx/nginx.conf
COPY --from=install-nginx /agent/nginx-agent.conf /etc/nginx-agent/nginx-agent.conf
RUN apt install -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold" -f /agent/build/$PACKAGE_NAME.deb

RUN chmod +x /agent/entrypoint.sh
STOPSIGNAL SIGTERM

EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]
