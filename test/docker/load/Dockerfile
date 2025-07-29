FROM ubuntu@sha256:a08e551cb33850e4740772b38217fc1796a66da2506d312abe51acda354ff061 AS base
LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

# https://askubuntu.com/questions/909277/avoiding-user-interaction-with-tzdata-when-installing-certbot-in-a-docker-contai
ARG DEBIAN_FRONTEND=noninteractive

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
                        jq \
    && apt-get remove --purge -y lsb-release \
    && apt-get remove --purge --auto-remove -y && rm -rf /var/lib/apt/lists/* /etc/apt/sources.list.d/nginx-plus.list \
    && rm -rf /etc/apt/apt.conf.d/90nginx /etc/ssl/nginx

EXPOSE 80

STOPSIGNAL SIGQUIT

CMD git config --global --add safe.directory /home/nginx

FROM base as configure-go

ARG GO_VERSION
ARG OSARCH

# Install Go
RUN wget https://go.dev/dl/go${GO_VERSION}.linux-${OSARCH}.tar.gz \
    && tar -xvf go${GO_VERSION}.linux-${OSARCH}.tar.gz \
    && mv go /usr/local

ENV DEBIAN_FRONTEND=noninteractive
ENV GOROOT=/usr/local/go
ENV CGO_ENABLED=0
ENV PATH="/usr/local/go/bin:${PATH}" 
ENV PATH=$PATH:/usr/local/go/bin

RUN mv /agent/test/config/agent/nginx-agent-otel-load.conf /agent/test/load/nginx-agent.conf
RUN mkdir /var/run/nginx-agent/ /var/log/nginx-agent/ /etc/nginx-agent/ 

WORKDIR /agent/
CMD make install-tools

FROM configure-go as run-tests

WORKDIR /agent/

RUN make build
RUN chmod -R +x ./build

WORKDIR /agent/test/load

RUN go test -v -timeout 1m ./...

CMD ["sh", "-c", "cat benchmarks.json"]
