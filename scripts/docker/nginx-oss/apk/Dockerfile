ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install-nginx
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

ARG ENTRY_POINT
ARG PACKAGE_NAME

WORKDIR /agent
COPY ./build /agent/build
COPY $ENTRY_POINT /agent/entrypoint.sh

# COPY ./test/integration/install/nginx.conf /etc/nginx/nginx.conf

RUN set -x \
    && addgroup -g 101 -S nginx \
    && adduser -S -D -H -u 101 -h /var/cache/nginx -s /sbin/nologin -G nginx -g nginx nginx \
    && apk add \
            ca-certificates \
            curl \
            bash \
            nginx

RUN chmod +x /agent/entrypoint.sh
STOPSIGNAL SIGTERM

EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]

FROM install-nginx as install-agent

ARG PACKAGE_NAME

RUN apk add --allow-untrusted /agent/build/$PACKAGE_NAME.apk
