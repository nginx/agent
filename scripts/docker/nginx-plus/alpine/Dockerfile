ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

ARG PACKAGES_REPO

WORKDIR /agent
COPY ./scripts/docker/nginx-plus/entrypoint.sh /agent/entrypoint.sh
COPY ./nginx-agent.conf /agent/nginx-agent.conf

RUN --mount=type=secret,id=nginx-crt,dst=/etc/apk/cert.pem \
    --mount=type=secret,id=nginx-key,dst=/etc/apk/cert.key \
    set -x \
    chmod 644 /etc/apk/cert* \
    # Create nginx user/group first, to be consistent throughout Docker variants
    && addgroup -g 101 -S nginx \
    && adduser -S -D -H -u 101 -h /var/cache/nginx -s /sbin/nologin -G nginx -g nginx nginx \
    # Check signing key
    && KEY_SHA512="e7fa8303923d9b95db37a77ad46c68fd4755ff935d0a534d26eba83de193c76166c68bfe7f65471bf8881004ef4aa6df3e34689c305662750c0172fca5d8552a *stdin" \
    && apk add --no-cache --virtual .cert-deps \
        openssl \
    && wget -O /tmp/nginx_signing.rsa.pub https://nginx.org/keys/nginx_signing.rsa.pub \
    && if [ "$(openssl rsa -pubin -in /tmp/nginx_signing.rsa.pub -text -noout | openssl sha512 -r)" = "$KEY_SHA512" ]; then \
        echo "key verification succeeded!"; \
        mv /tmp/nginx_signing.rsa.pub /etc/apk/keys/; \
    else \
        echo "key verification failed!"; \
        exit 1; \
    fi \
    && apk del .cert-deps \
    # Bring in gettext so we can get `envsubst`, then throw
    # the rest away. To do this, we need to install `gettext`
    # then move `envsubst` out of the way so `gettext` can
    # be deleted completely, then move `envsubst` back.
        && apk add --no-cache --virtual .gettext gettext \
        && mv /usr/bin/envsubst /tmp/ \
        \
        && runDeps="$( \
            scanelf --needed --nobanner /tmp/envsubst \
                | awk '{ gsub(/,/, "\nso:", $2); print "so:" $2 }' \
                | sort -u \
                | xargs -r apk info --installed \
                | sort -u \
        )" \
        && apk add --no-cache $runDeps \
        && apk del .gettext \
        && mv /tmp/envsubst /usr/local/bin/ \
        # Bring in tzdata so users could set the timezones through the environment
        # variables
        && apk add --no-cache tzdata \
        ## Optional: Install Tools
        # Bring in curl and ca-certificates to make registering on DNS SD easier
        && apk add --no-cache curl ca-certificates bash\ 
    # Prepare repo config and install NGINX Plus https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-plus/
    && wget -O /etc/apk/keys/nginx_signing.rsa.pub https://nginx.org/keys/nginx_signing.rsa.pub \
    && printf "https://$PACKAGES_REPO/plus/alpine/v`egrep -o '^[0-9]+\.[0-9]+' /etc/alpine-release`/main/\n" | tee -a /etc/apk/repositories \
    && printf "https://$PACKAGES_REPO/nginx-agent/alpine/v`egrep -o '^[0-9]+\.[0-9]+' /etc/alpine-release`/main/\n" | tee -a /etc/apk/repositories \
    && nginxPackages=" \
        nginx-plus \
        nginx-agent \
    " \
    && apk update \
    && apk add $nginxPackages

# run the nginx and agent
FROM install as runtime

COPY --from=install /agent/entrypoint.sh /agent/entrypoint.sh
COPY --from=install /agent/nginx-agent.conf /etc/nginx-agent/nginx-agent.conf

RUN chmod +x /agent/entrypoint.sh
STOPSIGNAL SIGTERM
EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]
