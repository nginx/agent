# Dockerfile for running NGINX Agent with NGINX Plus and rootless user
ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install-nginx-plus-rootless
LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

ARG DEBIAN_FRONTEND=noninteractive
ARG ENTRY_POINT
ARG PACKAGE_NAME
ARG PACKAGES_REPO

WORKDIR /agent
COPY ./build /agent/build
COPY $ENTRY_POINT /agent/entrypoint.sh
RUN chmod +x /agent/entrypoint.sh

ENV PLUS_VERSION=R32

RUN --mount=type=secret,id=nginx-crt,dst=nginx-repo.crt \
    --mount=type=secret,id=nginx-key,dst=nginx-repo.key \
    set -x \
# Create nginx user/group first, to be consistent throughout Docker variants
    && groupadd --system --gid 101 nginx \
    && useradd --system --gid nginx --no-create-home --home-dir /nonexistent --uid 101 nginx \
    && apt-get update \
    && apt-get install --no-install-recommends --no-install-suggests -y \
                        ca-certificates \
                        gpg \
                        lsb-release \
                        git \
                        wget \
                        make \
# Install the latest release of NGINX Plus and/or NGINX Plus modules
# Uncomment individual modules if necessary
# Use versioned packages over defaults to specify a release
    && nginxPackages=" \
        nginx-plus \
    " \
    && echo "Acquire::https::pkgs.nginx.com::Verify-Peer \"true\";" > /etc/apt/apt.conf.d/90nginx \
    && echo "Acquire::https::pkgs.nginx.com::Verify-Host \"true\";" >> /etc/apt/apt.conf.d/90nginx \
    && echo "Acquire::https::pkgs.nginx.com::SslCert     \"/etc/ssl/nginx/nginx-repo.crt\";" >> /etc/apt/apt.conf.d/90nginx \
    && echo "Acquire::https::pkgs.nginx.com::SslKey      \"/etc/ssl/nginx/nginx-repo.key\";" >> /etc/apt/apt.conf.d/90nginx \
    && printf "deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] https://pkgs.nginx.com/plus/${PLUS_VERSION}/ubuntu/ `lsb_release -cs` nginx-plus\n" > /etc/apt/sources.list.d/nginx-plus.list \
    && wget -qO - https://cs.nginx.com/static/keys/nginx_signing.key | gpg --dearmor | tee /usr/share/keyrings/nginx-archive-keyring.gpg >/dev/null \
    && mkdir -p /etc/ssl/nginx \
    && cat nginx-repo.crt > /etc/ssl/nginx/nginx-repo.crt \
    && cat nginx-repo.key > /etc/ssl/nginx/nginx-repo.key \
    && apt-get update \
    && apt-get install --no-install-recommends --no-install-suggests -y \
                        $nginxPackages \
                        curl \
                        gettext-base \
                        jq \
                        gnupg2 \
    && apt-get remove --purge -y lsb-release \
    && apt-get remove --purge --auto-remove -y && rm -rf /var/lib/apt/lists/* /etc/apt/sources.list.d/nginx-plus.list \
    && rm -rf /etc/apt/apt.conf.d/90nginx /etc/ssl/nginx 


# Copy custom nginx config for rootless setup
COPY ./test/config/nginx/nginx-rootless-plus.conf /etc/nginx/nginx.conf
RUN chown nginx:nginx /etc/nginx/nginx.conf \
    && chmod 644 /etc/nginx/nginx.conf

# Create necessary directories and set permissions for nginx user
RUN mkdir -p /var/log/nginx /var/cache/nginx /var/run/nginx /var/lib/nginx /etc/nginx/conf.d /var/log/nginx-agent /var/log/app_protect /etc/nginx-agent /etc/nginx/ssl \
    && chown -R nginx:nginx /var/log/nginx /var/cache/nginx /var/run/nginx /var/lib/nginx /etc/nginx /var/log/nginx-agent /var/log/app_protect /etc/nginx-agent /etc/nginx/ssl \
    && chmod 755 /var/log/nginx /var/cache/nginx /var/run/nginx /var/lib/nginx /etc/nginx /var/log/nginx-agent /etc/nginx-agent /etc/nginx/ssl \
    && touch /var/lib/nginx/nginx.id || true

# Grant CAP_NET_BIND_SERVICE so nginx (running non-root) can bind privileged ports
RUN apt-get update && apt-get install -y --no-install-recommends libcap2-bin \
    && setcap 'cap_net_bind_service=+ep' /usr/sbin/nginx \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

RUN apt-get update && apt-get install -y /agent/build/$PACKAGE_NAME.deb \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Fix permissions for nginx agent config and directories
RUN if [[ -f /etc/nginx-agent/nginx-agent.conf ]]; then chmod 644 /etc/nginx-agent/nginx-agent.conf; fi \
    && chmod 755 /etc/nginx-agent \
    && chown -R nginx:nginx /etc/nginx-agent /var/log/nginx-agent /var/lib/nginx-agent /var/lib/nginx /var/run/nginx

STOPSIGNAL SIGTERM

# Expose ports (80 for HTTP with capability, 443 for HTTPS with capability)
EXPOSE 80 443

# Switch to nginx user
USER nginx

WORKDIR /etc/nginx-agent

ENTRYPOINT ["/agent/entrypoint.sh"]
