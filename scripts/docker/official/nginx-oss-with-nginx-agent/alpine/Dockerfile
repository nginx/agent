ARG NGINX_VERSION=1.25.1-alpine
FROM nginx:${NGINX_VERSION}
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

RUN set -x \
    && apk add curl bash iproute2

RUN printf "%s%s\n" \
    "http://packages.nginx.org/nginx-agent/alpine/v3.16" \
    "/main" | tee -a /etc/apk/repositories

RUN curl -o /etc/apk/keys/nginx_signing.rsa.pub https://nginx.org/keys/nginx_signing.rsa.pub

RUN apk add nginx-agent

COPY ./scripts/docker/entrypoint.sh /agent/entrypoint.sh
RUN chmod +x /agent/entrypoint.sh

STOPSIGNAL SIGTERM

ENTRYPOINT ["/agent/entrypoint.sh"]
