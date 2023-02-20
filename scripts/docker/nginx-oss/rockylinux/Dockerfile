ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install-nginx
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

ARG NGINX_CONF
ARG ENTRY_POINT

WORKDIR /agent
COPY ./ /agent
COPY $ENTRY_POINT /agent/entrypoint.sh

RUN set -x \
    && groupadd --system --gid 101 nginx \
    && adduser -g nginx --system --no-create-home --home /nonexistent --shell /bin/false --uid 101 nginx \
    && usermod -s /sbin/nologin nginx \
    && usermod -L nginx \
    && yum install -y git \
                      wget \
                      procps \
                      make \
                      vim \
                      nginx

RUN chmod +x /agent/entrypoint.sh
STOPSIGNAL SIGTERM

EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]


FROM install-nginx as install-agent

ARG PACKAGE_NAME
ARG AGENT_CONF

COPY $AGENT_CONF /agent/nginx-agent.conf
COPY --from=install-nginx /agent/nginx-agent.conf /etc/nginx-agent/nginx-agent.conf
RUN yum localinstall -y /agent/build/$PACKAGE_NAME.rpm
