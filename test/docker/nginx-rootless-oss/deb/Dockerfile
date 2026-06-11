# Dockerfile for running NGINX Agent with rootless user
ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install-nginx-rootless
LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

ARG DEBIAN_FRONTEND=noninteractive
ARG ENTRY_POINT
ARG PACKAGE_NAME
ARG PACKAGES_REPO

WORKDIR /agent
COPY ./build /agent/build
COPY $ENTRY_POINT /agent/entrypoint.sh
RUN chmod +x /agent/entrypoint.sh

# Install system dependencies and create nginx user
RUN set -x \
    && groupadd --system --gid 101 nginx \
    && useradd --system --gid nginx --no-create-home --home /nonexistent --comment "nginx user" --shell /bin/false --uid 101 nginx \
    && apt-get update \
    && apt-get install --no-install-recommends --no-install-suggests -y \
        ca-certificates \
        gnupg2 \
        curl \
        lsb-release \
        procps \
        nginx \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Create necessary directories and set permissions for nginx user
RUN mkdir -p /var/log/nginx /var/cache/nginx /var/run/nginx /var/lib/nginx /etc/nginx/conf.d /etc/nginx/ssl \
    && chown -R nginx:nginx /var/log/nginx /var/cache/nginx /var/run/nginx /var/lib/nginx /etc/nginx  /etc/nginx/ssl \
    && chmod 755 /var/log/nginx /var/cache/nginx /var/run/nginx /var/lib/nginx /etc/nginx /etc/nginx/ssl \
    && touch /var/lib/nginx/nginx.id || true

# Copy custom nginx config for rootless setup
COPY ./test/config/nginx/nginx-rootless-oss.conf /etc/nginx/nginx.conf
RUN chown nginx:nginx /etc/nginx/nginx.conf \
    && chmod 644 /etc/nginx/nginx.conf

# Grant CAP_NET_BIND_SERVICE to allow binding to privileged ports as non-root
RUN apt-get update && apt-get install -y libcap2-bin \
    && setcap 'cap_net_bind_service=+ep' /usr/sbin/nginx \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

STOPSIGNAL SIGTERM

# Expose ports (80 for HTTP with capability, 443 for HTTPS with capability)
EXPOSE 80 443

# Switch to nginx user
USER nginx

WORKDIR /etc/nginx-agent
ENTRYPOINT ["/agent/entrypoint.sh"]

# Install agent from local package
FROM install-nginx-rootless as install-agent-local

ARG PACKAGE_NAME

USER root

RUN apt-get update && apt-get install -y /agent/build/$PACKAGE_NAME.deb \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Fix permissions for nginx agent config and directories
RUN if [[ -f /etc/nginx-agent/nginx-agent.conf ]]; then chmod 644 /etc/nginx-agent/nginx-agent.conf; fi \
    && chmod 755 /etc/nginx-agent /var/log/nginx-agent \
    && chown -R nginx:nginx /etc/nginx-agent /var/log/nginx-agent /var/lib/nginx-agent

USER nginx

# Install agent from repository
FROM install-nginx-rootless as install-agent-repo

USER root

RUN curl https://nginx.org/keys/nginx_signing.key | gpg --dearmor | tee /usr/share/keyrings/nginx-archive-keyring.gpg >/dev/null \
    && printf "deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] http://packages.nginx.org/nginx-agent/ubuntu/ `lsb_release -cs` agent\n" > /etc/apt/sources.list.d/nginx-agent.list

RUN apt-get update && apt-get install -y nginx-agent \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Fix permissions for nginx agent config and directories
RUN if [[ -f /etc/nginx-agent/nginx-agent.conf ]]; then chmod 644 /etc/nginx-agent/nginx-agent.conf; fi \
    && chmod 755 /etc/nginx-agent /var/log/nginx-agent \
    && chown -R nginx:nginx /etc/nginx-agent /var/log/nginx-agent /var/lib/nginx-agent

USER nginx
