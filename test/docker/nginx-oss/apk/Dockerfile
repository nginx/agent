ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install-nginx
LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

ARG ENTRY_POINT
ARG PACKAGES_REPO

WORKDIR /agent
COPY $ENTRY_POINT /agent/entrypoint.sh

RUN set -x \
    && addgroup -g 101 -S nginx \
    && adduser -S -D -H -u 101 -h /var/cache/nginx -s /sbin/nologin -G nginx -g nginx nginx \
    && apk add ca-certificates \
               openrc \
               curl \
               openssl \
               bash \
               nginx

## Allow openrc to run in a container
RUN mkdir /run/openrc \
    && touch /run/openrc/softlevel && echo 'rc_need="!dev !net"' >> /etc/rc.conf

RUN chmod +x /agent/entrypoint.sh
STOPSIGNAL SIGTERM

EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]

FROM install-nginx as install-agent-local

ARG PACKAGE_NAME

RUN apk add --allow-untrusted /agent/build/${PACKAGE_NAME}.apk

FROM install-nginx as install-agent-repo

# Setup nginx agent repository
RUN curl -o /etc/apk/keys/nginx_signing.rsa.pub https://nginx.org/keys/nginx_signing.rsa.pub

RUN printf "%s\n" \
    "http://packages.nginx.org/nginx-agent/alpine/v3.22/main" \
      >> /etc/apk/repositories && apk update

RUN apk add nginx-agent=3.2.1 && rc-update add nginx-agent default
