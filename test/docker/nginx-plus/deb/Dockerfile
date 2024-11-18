# Base Image Argument
ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install-nginx
LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

# Define arguments for flexibility
ARG DEBIAN_FRONTEND=noninteractive
ARG ENTRY_POINT
ARG PACKAGE_NAME
ARG PACKAGES_REPO

# Set working directory
WORKDIR /agent
COPY ./build /agent/build
COPY $ENTRY_POINT /agent/entrypoint.sh

# Install NGINX Plus and NGINX App Protect
RUN --mount=type=secret,id=nginx-crt,dst=nginx-repo.crt \
    --mount=type=secret,id=nginx-key,dst=nginx-repo.key \
    set -eux \
    && groupadd --system --gid 101 nginx \
    && useradd --system --gid nginx --no-create-home --home-dir /nonexistent --uid 101 nginx \
    \
    && apt-get update \
    && apt-get install --no-install-recommends --no-install-suggests -y \
        ca-certificates \
        gnupg2 \
        lsb-release \
        git \
        wget \
        make \
        apt-transport-https \
        ubuntu-keyring \
    \
    && mkdir -p /etc/ssl/nginx \
    && cat nginx-repo.crt > /etc/ssl/nginx/nginx-repo.crt \
    && cat nginx-repo.key > /etc/ssl/nginx/nginx-repo.key \
    \
    # Add NGINX repositories securely
    && wget -qO - https://cs.nginx.com/static/keys/nginx_signing.key | gpg --dearmor > /usr/share/keyrings/nginx-archive-keyring.gpg \
    && wget -qO - https://cs.nginx.com/static/keys/app-protect-security-updates.key | gpg --dearmor > /usr/share/keyrings/app-protect-security-updates.gpg \
    \
    && printf "deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] https://pkgs.nginx.com/plus/ubuntu $(lsb_release -cs) nginx-plus\n" > /etc/apt/sources.list.d/nginx-plus.list \
    && printf "deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] https://pkgs.nginx.com/app-protect/ubuntu $(lsb_release -cs) nginx-plus\n" > /etc/apt/sources.list.d/nginx-app-protect.list \
    && printf "deb [signed-by=/usr/share/keyrings/app-protect-security-updates.gpg] https://pkgs.nginx.com/app-protect-security-updates/ubuntu $(lsb_release -cs) nginx-plus\n" >> /etc/apt/sources.list.d/nginx-app-protect.list \
    \
    && wget -P /etc/apt/apt.conf.d https://cs.nginx.com/static/files/90pkgs-nginx \
    \
    && apt-get update \
    && apt-cache show nginx-plus || echo "nginx-plus package not found" \
    && apt-cache show app-protect || echo "app-protect package not found" \
    && apt-get install --no-install-recommends --no-install-suggests -y \
        nginx-plus \
        app-protect \
        app-protect-attack-signatures \
        curl \
        gettext-base \
        jq \
    \
    && apt-get remove --purge -y lsb-release \
    && apt-get remove --purge --auto-remove -y \
    \
    # Clean up
    && rm -rf /var/lib/apt/lists/* \
              /etc/apt/sources.list.d/nginx-plus.list \
              /etc/apt/sources.list.d/nginx-app-protect.list \
              /etc/apt/apt.conf.d/90nginx \
              /etc/ssl/nginx

# Expose ports
EXPOSE 80 443

# Set stop signal
STOPSIGNAL SIGQUIT

# Set executable permissions for entrypoint
RUN chmod +x /agent/entrypoint.sh

# Install NGINX Agent package
RUN apt install -y /agent/build/${PACKAGE_NAME}.deb || { echo "Failed to install ${PACKAGE_NAME}"; exit 1; }

# Define entrypoint
ENTRYPOINT ["/agent/entrypoint.sh"]
