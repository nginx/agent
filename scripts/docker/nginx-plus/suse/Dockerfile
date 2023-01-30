ARG OS_RELEASE
ARG OS_VERSION

FROM registry.suse.com/${OS_RELEASE}/${OS_VERSION} as install
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

ARG OS_VERSION
ARG PACKAGES_REPO
ARG BASE_IMAGE

WORKDIR /agent
COPY ./scripts/docker/nginx-plus/entrypoint.sh /agent/entrypoint.sh
COPY ./nginx-agent.conf /agent/nginx-agent.conf

RUN --mount=type=secret,id=nginx-crt,dst=/nginx-repo.crt \
    --mount=type=secret,id=nginx-key,dst=/nginx-repo.key \
    set -x \
    && mkdir -p /etc/ssl/nginx \
    && cat /nginx-repo.crt > /etc/ssl/nginx/nginx-repo.crt \
    && cat /nginx-repo.key > /etc/ssl/nginx/nginx-repo.key \
    && cat /etc/ssl/nginx/nginx-repo.crt /etc/ssl/nginx/nginx-repo.key > /etc/ssl/nginx/nginx-repo-bundle.crt \
    && zypper install ca-certificates \
    && nginxPackages=" \
        nginx-plus \
        nginx-agent \
       " \
    &&  if [ "$OS_VERSION" = "sles12sp5" ]; then \
            zypper addrepo -G -c "https://$PACKAGES_REPO/plus/sles/12?ssl_clientcert=/etc/ssl/nginx/nginx-repo-bundle.crt&ssl_verify=peer" nginx-plus \
            && zypper addrepo -G -c "https://$PACKAGES_REPO/nginx-agent/sles/12?ssl_clientcert=/etc/ssl/nginx/nginx-repo-bundle.crt&ssl_verify=peer" nginx-agent; \
        else \
            zypper addrepo -G -c "https://$PACKAGES_REPO/plus/sles/15?ssl_clientcert=/etc/ssl/nginx/nginx-repo-bundle.crt&ssl_verify=peer" nginx-plus \
            && zypper addrepo -G -c "https://$PACKAGES_REPO/nginx-agent/sles/15?ssl_clientcert=/etc/ssl/nginx/nginx-repo-bundle.crt&ssl_verify=peer" nginx-agent; \
        fi \
    && zypper refresh \
    && zypper install -y $nginxPackages \
    && rm -rf /etc/ssl/nginx/nginx-repo-bundle.crt /etc/ssl/nginx

FROM install as runtime

COPY --from=install /agent/entrypoint.sh /agent/entrypoint.sh
COPY --from=install /agent/nginx-agent.conf /etc/nginx-agent/nginx-agent.conf

RUN chmod +x /agent/entrypoint.sh
STOPSIGNAL SIGTERM
EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]
