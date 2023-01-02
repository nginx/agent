ARG DOCKER_IMAGE
FROM ${DOCKER_IMAGE} as install
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

ARG PACKAGES_REPO

RUN echo $PACKAGES_REPO
WORKDIR /agent
COPY ./scripts/docker/nginx-plus/entrypoint.sh /agent/entrypoint.sh
COPY ./nginx-agent.conf /agent/nginx-agent.conf

RUN --mount=type=secret,id=nginx-crt,dst=/nginx-repo.crt \
    --mount=type=secret,id=nginx-key,dst=/nginx-repo.key \
    set -x \
    && addgroup --system --gid 101 nginx \
    && adduser --system --disabled-login --ingroup nginx --no-create-home --home /nonexistent --gecos "nginx user" --shell /bin/false --uid 101 nginx \
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
    # Install the latest release of NGINX Plus and/or NGINX Plus modules and NGINX Agent
    && nginxPackages=" \
        nginx-plus \
        nginx-agent \
    " \
    && echo "Acquire::https::$PACKAGES_REPO::Verify-Peer \"true\";" > /etc/apt/apt.conf.d/90nginx \
    && echo "Acquire::https::$PACKAGES_REPO::Verify-Host \"true\";" >> /etc/apt/apt.conf.d/90nginx \
    && echo "Acquire::https::$PACKAGES_REPO::SslCert     \"/etc/ssl/nginx/nginx-repo.crt\";" >> /etc/apt/apt.conf.d/90nginx \
    && echo "Acquire::https::$PACKAGES_REPO::SslKey      \"/etc/ssl/nginx/nginx-repo.key\";" >> /etc/apt/apt.conf.d/90nginx \
    && apt-get install apt-transport-https lsb-release ca-certificates \
    && apt-cache policy | awk '{print $2" "$3}' | sort -u \
    && printf "deb https://$PACKAGES_REPO/plus/ubuntu/ `lsb_release -cs` nginx-plus\n" > /etc/apt/sources.list.d/nginx-plus.list \
    && printf "deb https://$PACKAGES_REPO/nginx-agent/ubuntu/ `lsb_release -cs` nginx-plus\n" > /etc/apt/sources.list.d/nginx-agent.list \
    && mkdir -p /etc/ssl/nginx \
    && cat /nginx-repo.crt > /etc/ssl/nginx/nginx-repo.crt \
    && cat /nginx-repo.key > /etc/ssl/nginx/nginx-repo.key \
    && apt-get update \
    && apt-get install $nginxPackages -y  \
    && rm /etc/ssl/nginx/nginx-repo.crt /etc/ssl/nginx/nginx-repo.key

# run the nginx and agent
FROM install as runtime

COPY --from=install /agent/entrypoint.sh /agent/entrypoint.sh
COPY --from=install /agent/nginx-agent.conf /etc/nginx-agent/nginx-agent.conf

RUN chmod +x /agent/entrypoint.sh
STOPSIGNAL SIGTERM
EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]
