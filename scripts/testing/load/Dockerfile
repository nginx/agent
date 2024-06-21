FROM ubuntu:22.04

LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

# Define NGINX versions for NGINX Plus and NGINX Plus modules
# Uncomment this block and the versioned nginxPackages block in the main RUN
# instruction to install a specific release
# ENV NGINX_VERSION   25
# ENV NJS_VERSION     0.7.0
# ENV PKG_RELEASE     1~bullseye

# Download certificate and key from the customer portal (https://account.f5.com)
# and copy to the build context

# https://askubuntu.com/questions/909277/avoiding-user-interaction-with-tzdata-when-installing-certbot-in-a-docker-contai
ARG DEBIAN_FRONTEND=noninteractive
ARG ENTRY_POINT
ARG PACKAGE_NAME
ARG PACKAGES_REPO
ARG GO_VERSION
ARG OSARCH

WORKDIR /agent
COPY . /agent

RUN --mount=type=secret,id=nginx-crt,dst=nginx-repo.crt \
    --mount=type=secret,id=nginx-key,dst=nginx-repo.key \
    set -x \
# Create nginx user/group first, to be consistent throughout Docker variants
    && groupadd --system --gid 101 nginx \
    && useradd --system --gid nginx --no-create-home --home-dir /nonexistent --uid 101 nginx \
    && apt-get update \
    && apt-get install --no-install-recommends --no-install-suggests -y \
                        ca-certificates \
                        gnupg1 \
                        lsb-release \
                        git \
                        wget \
                        make \
    && \
    NGINX_GPGKEY=573BFD6B3D8FBC641079A6ABABF5BD827BD9BF62; \
    found=''; \
    for server in \
        hkp://keyserver.ubuntu.com:80 \
        pgp.mit.edu \
    ; do \
        echo "Fetching GPG key $NGINX_GPGKEY from $server"; \
        apt-key adv --keyserver "$server" --keyserver-options timeout=10 --recv-keys "$NGINX_GPGKEY" && found=yes && break; \
    done; \
    test -z "$found" && echo >&2 "error: failed to fetch GPG key $NGINX_GPGKEY" && exit 1; \
    apt-get remove --purge --auto-remove -y gnupg1 && rm -rf /var/lib/apt/lists/* \
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
    && printf "deb https://pkgs.nginx.com/plus/ubuntu `lsb_release -cs` nginx-plus\n" > /etc/apt/sources.list.d/nginx-plus.list \
    && mkdir -p /etc/ssl/nginx \
    && cat nginx-repo.crt > /etc/ssl/nginx/nginx-repo.crt \
    && cat nginx-repo.key > /etc/ssl/nginx/nginx-repo.key \
    && apt-get update \
    && apt-get install --no-install-recommends --no-install-suggests -y \
                        $nginxPackages \
                        curl \
                        gettext-base \
    && apt-get remove --purge -y lsb-release \
    && apt-get remove --purge --auto-remove -y && rm -rf /var/lib/apt/lists/* /etc/apt/sources.list.d/nginx-plus.list \
    && rm -rf /etc/apt/apt.conf.d/90nginx /etc/ssl/nginx

EXPOSE 80

STOPSIGNAL SIGQUIT

RUN git config --global --add safe.directory /home/nginx

# Install Go
RUN wget https://go.dev/dl/go${GO_VERSION}.linux-${OSARCH}.tar.gz \
    && tar -xvf go${GO_VERSION}.linux-${OSARCH}.tar.gz \
    && mv go /usr/local

ENV GOROOT=/usr/local/go
ENV CGO_ENABLED=0
ENV PATH="/usr/local/go/bin:${PATH}" 
ENV PACKAGE_PATH="/agent/build/${PACKAGE_NAME}.deb"

RUN mv /agent/test/config/agent/nginx-agent-otel-load.conf /agent/test/load/nginx-agent.conf

# Run benchmark tests
# RUN mkdir -p /data/nginx/
# ADD test/testdata/configs/bigger/metrics.conf /etc/nginx/nginx.conf
# WORKDIR "/home/nginx"
CMD cd /agent/test/load && go test -timeout 30s -run ^TestMetric10kDPS$$ github.com/nginx/agent/v3/test/load

# CMD mv -R /agent/test/load/results /agent/build/results && mv /agent/test/load/benchmarks.json /agent/build/results

# ARG BASE_IMAGE
# FROM ubuntu:22.04 as install-nginx
# LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

# ARG DEBIAN_FRONTEND=noninteractive
# ARG ENTRY_POINT
# ARG PACKAGE_NAME
# ARG PACKAGES_REPO

# WORKDIR /agent
# COPY ./build /agent/build
# COPY ./test/load /agent/test/load 
# COPY ./test/config/agent/nginx-agent-otel-load.conf /agent/nginx-agent.conf

# # COPY $ENTRY_POINT /agent/entrypoint.sh

# RUN set -x \
#     && groupadd --system --gid 101 nginx \
#     && useradd --system --gid nginx --no-create-home --home-dir /nonexistent --uid 101 nginx \
#     && apt-get update \
#     && add-apt-repository ppa:longsleep/golang-backports \
#     && apt-get update \
#     && apt-get install --no-install-recommends --no-install-suggests -y ca-certificates \
#                                                                         gnupg2 \
#                                                                         git \
#                                                                         make \
#                                                                         curl \
#                                                                         lsb-release \
#                                                                         procps \
#                                                                         nginx \
#                                                                         golang-go
# # Setup nginx agent repository

# # RUN chmod +x /agent/entrypoint.sh
# STOPSIGNAL SIGTERM

# EXPOSE 80 443

# CMD CGO_ENABLED=0 PACKAGE_PATH=$PACKAGE_NAME go test -timeout 30s -run ^TestMetric10kDPS$$ github.com/nginx/agent/v3/test/load