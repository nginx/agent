ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install-nginx
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

ARG NGINX_CONF
ARG AGENT_CONF
ARG ENTRY_POINT

WORKDIR /agent
COPY ./ /agent
COPY $ENTRY_POINT /agent/entrypoint.sh

RUN if [[ -z "$AGENT_CONF" ]] ; then cp $AGENT_CONF /etc/nginx-agent/nginx-agent.conf ; fi
RUN if [[ -z "$NGINX_CONF" ]] ; then cp $NGINX_CONF /etc/nginx/nginx.conf ; fi

RUN set -x \
    && addgroup --system --gid 101 nginx \
    && adduser --system --disabled-login --ingroup nginx --no-create-home --home /nonexistent --gecos "nginx user" --shell /bin/false --uid 101 nginx \
    && apt-get update \
    && apt-get install --no-install-recommends --no-install-suggests -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold" \
                        ca-certificates \
                        gnupg1 \
                        lsb-release \
                        git \
                        wget \
                        make \
                        curl \
                        vim \
                        nginx
RUN chmod +x /agent/entrypoint.sh
STOPSIGNAL SIGTERM

EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]

FROM install-nginx as install-agent

ARG PACKAGE_NAME
ARG AGENT_CONF

COPY --from=install-nginx /agent/nginx-agent.conf /etc/nginx-agent/nginx-agent.conf
RUN apt install -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold" -f /agent/build/$PACKAGE_NAME.deb
