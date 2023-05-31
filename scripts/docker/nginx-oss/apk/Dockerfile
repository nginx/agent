ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install-nginx
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

ARG ENTRY_POINT
ARG PACKAGE_NAME

WORKDIR /agent
COPY ./build /agent/build
COPY $ENTRY_POINT /agent/entrypoint.sh

RUN set -x \
    && addgroup -g 101 -S nginx \
    && adduser -S -D -H -u 101 -h /var/cache/nginx -s /sbin/nologin -G nginx -g nginx nginx \
    && apk add ca-certificates \
               curl \
               bash \
               nginx

RUN chmod +x /agent/entrypoint.sh
STOPSIGNAL SIGTERM

EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]


FROM install-nginx as install-agent-local

ARG PACKAGE_NAME

RUN apk add --allow-untrusted /agent/build/${PACKAGE_NAME}.apk


FROM install-nginx as install-agent-repo

ARG PACKAGES_REPO

RUN printf "%s%s%s%s\n" \
    "@nginx-agent " \
    "http://${PACKAGES_REPO}/nginx-agent/alpine/v" \
    `grep -o -E '^[0-9]+\.[0-9]+' /etc/alpine-release` \
    "/main" | tee -a /etc/apk/repositories

RUN curl -o /etc/apk/keys/nginx_signing.rsa.pub https://nginx.org/keys/nginx_signing.rsa.pub
    
RUN apk add nginx-agent@nginx-agent

