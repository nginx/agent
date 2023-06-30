ARG NGINX_VERSION=1.25.1-alpine
FROM nginx:${NGINX_VERSION}
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

# Unlink symbolic links for access and error logs as the NGINX Agent 
# requires both files to be able to monitor NGINX and provide metrics.
RUN unlink /var/log/nginx/access.log && unlink /var/log/nginx/error.log

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
