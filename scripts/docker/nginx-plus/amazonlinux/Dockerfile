ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

ARG BASE_IMAGE
ARG PACKAGES_REPO
# note if OS_VERSION is 1, there will be an issues in the yum.repos.d folder
ARG OS_VERSION

WORKDIR /agent
COPY ./scripts/docker/nginx-plus/entrypoint.sh /agent/entrypoint.sh
COPY ./nginx-agent.conf /agent/nginx-agent.conf

RUN --mount=type=secret,id=nginx-crt,dst=/nginx-repo.crt \
    --mount=type=secret,id=nginx-key,dst=/nginx-repo.key \
    set -x \
    && mkdir -p /etc/ssl/nginx \
    && cat /nginx-repo.crt > /etc/ssl/nginx/nginx-repo.crt \
    && cat /nginx-repo.key > /etc/ssl/nginx/nginx-repo.key \
    && yum install -y --setopt=tsflags=nodocs wget ca-certificates bind-utils wget bind-utils vim-minimal shadow-utils \
    && groupadd --system --gid 101 nginx \
    && adduser -g nginx --system --no-create-home --home /nonexistent --shell /bin/false --uid 101 nginx \
    && nginxPackages=" \
        nginx-plus \
        nginx-agent \
       " \
    && printf "[nginx-plus] \n\
name=nginx-plus repo \n\
baseurl=https://$PACKAGES_REPO/plus/amzn$OS_VERSION/\$releasever/\$basearch \n\
sslclientcert=/etc/ssl/nginx/nginx-repo.crt \n\
sslclientkey=/etc/ssl/nginx/nginx-repo.key \n\
gpgcheck=1 \n\
enabled=1 \n\
sslverify=true \n\
gpgkey=https://nginx.org/keys/nginx_signing.key" >> /etc/yum.repos.d/nginx-plus-$BASE_IMAGE.repo \
    && printf "[nginx-agent] \n\
name=nginx-agent repo \n\
baseurl=https://$PACKAGES_REPO/nginx-agent/amzn$OS_VERSION/\$releasever/\$basearch \n\
sslclientcert=/etc/ssl/nginx/nginx-repo.crt \n\
sslclientkey=/etc/ssl/nginx/nginx-repo.key \n\
gpgcheck=1 \n\
enabled=1 \n\
sslverify=true \n\
gpgkey=https://nginx.org/keys/nginx_signing.key" >> /etc/yum.repos.d/nginx-agent-$BASE_IMAGE.repo \
    && yum install -y $nginxPackages \
    && yum clean all \
    && rm -rf /var/cache/yum /etc/yum.repos.d/* /etc/ssl/nginx

FROM install as runtime

COPY --from=install /agent/entrypoint.sh /agent/entrypoint.sh
COPY --from=install /agent/nginx-agent.conf /etc/nginx-agent/nginx-agent.conf

RUN chmod +x /agent/entrypoint.sh
STOPSIGNAL SIGTERM
EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]
