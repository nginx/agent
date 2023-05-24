ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

ARG PACKAGES_REPO

RUN echo $PACKAGES_REPO
WORKDIR /agent
COPY ./scripts/docker/entrypoint.sh /agent/entrypoint.sh
COPY ./nginx-agent.conf /agent/nginx-agent.conf

RUN --mount=type=secret,id=nginx-crt,dst=/nginx-repo.crt \
    --mount=type=secret,id=nginx-key,dst=/nginx-repo.key \
    set -x \
    && addgroup --system --gid 101 nginx \
    && adduser --system --disabled-login --ingroup nginx --no-create-home --home /nonexistent --gecos "nginx user" --shell /bin/false --uid 101 nginx \
    && apt-get update \
    && apt-get install --no-install-recommends --no-install-suggests -y \
                        ca-certificates \
                        gnupg2 \
                        lsb-release \
                        git \
                        wget \
                        make \
                        apt-transport-https \
    && \
    curl https://nginx.org/keys/nginx_signing.key | gpg --dearmor | tee /usr/share/keyrings/nginx-archive-keyring.gpg >/dev/null \
    # Install the latest release of NGINX Plus and/or NGINX Plus modules and NGINX Agent
    && nginxPackages=" \
        nginx-plus \
        nginx-agent \
    " \
    && echo "Acquire::https::$PACKAGES_REPO::Verify-Peer \"true\";" > /etc/apt/apt.conf.d/90nginx \
    && echo "Acquire::https::$PACKAGES_REPO::Verify-Host \"true\";" >> /etc/apt/apt.conf.d/90nginx \
    && echo "Acquire::https::$PACKAGES_REPO::SslCert     \"/etc/ssl/nginx/nginx-repo.crt\";" >> /etc/apt/apt.conf.d/90nginx \
    && echo "Acquire::https::$PACKAGES_REPO::SslKey      \"/etc/ssl/nginx/nginx-repo.key\";" >> /etc/apt/apt.conf.d/90nginx \
    && apt-cache policy | awk '{print $2" "$3}' | sort -u \
    && printf "deb https://$PACKAGES_REPO/plus/ubuntu/ `lsb_release -cs` nginx-plus\n" > /etc/apt/sources.list.d/nginx-plus.list \
    && printf "deb https://$PACKAGES_REPO/nginx-agent/ubuntu/ `lsb_release -cs` nginx-plus\n" > /etc/apt/sources.list.d/nginx-agent.list \
    && mkdir -p /etc/ssl/nginx \
    && cp /nginx-repo.crt /etc/ssl/nginx/nginx-repo.crt \
    && cp /nginx-repo.key /etc/ssl/nginx/nginx-repo.key \
    && apt-get update \
    && apt-get install $nginxPackages -y  \
    && rm /etc/ssl/nginx/nginx-repo.crt /etc/ssl/nginx/nginx-repo.key /nginx-repo.crt /nginx-repo.key

# run the nginx and agent
FROM install as runtime

COPY --from=install /agent/entrypoint.sh /agent/entrypoint.sh
COPY --from=install /agent/nginx-agent.conf /etc/nginx-agent/nginx-agent.conf

RUN chmod +x /agent/entrypoint.sh
STOPSIGNAL SIGTERM
EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]
