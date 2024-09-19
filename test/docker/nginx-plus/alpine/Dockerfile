ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install
LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

ARG PACKAGES_REPO
ARG NGINX_AGENT_VERSION

WORKDIR /agent
COPY ./test/docker/entrypoint.sh /agent/entrypoint.sh
COPY ./nginx-agent.conf /agent/nginx-agent.conf

RUN --mount=type=secret,id=nginx-crt,dst=/etc/apk/cert.pem \
    --mount=type=secret,id=nginx-key,dst=/etc/apk/cert.key \
    set -x \
    chmod 644 /etc/apk/cert* \
    # Create nginx user/group first, to be consistent throughout Docker variants
    && addgroup -g 101 -S nginx \
    && adduser -S -D -H -u 101 -h /var/cache/nginx -s /sbin/nologin -G nginx -g nginx nginx \
    # Check signing key
    && KEY_SHA512="de7031fdac1354096d3388d6f711a508328ce66c168967ee0658c294226d6e7a161ce7f2628d577d56f8b63ff6892cc576af6f7ef2a6aa2e17c62ff7b6bf0d98 *stdin" \
    && apk add --no-cache --virtual .cert-deps \
        openssl \
    && wget -O /tmp/nginx_signing.rsa.pub https://nginx.org/keys/nginx_signing.rsa.pub \
    # sed replace is required for openssl=v1.x which is used in alpine<=3.16
    && if [ "$(openssl rsa -pubin -in /tmp/nginx_signing.rsa.pub -text -noout | sed -e 's/RSA Public-Key/Public-Key/' | openssl sha512 -r)" = "$KEY_SHA512" ]; then \
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
    " \
    && apk update \
    && apk add $nginxPackages

RUN if [ -z "$NGINX_AGENT_VERSION" ]; then apk add nginx-agent; else apk add "nginx-agent=${NGINX_AGENT_VERSION}"; fi

# run the nginx and agent
FROM install as runtime

COPY --from=install /agent/entrypoint.sh /agent/entrypoint.sh
COPY --from=install /agent/nginx-agent.conf /etc/nginx-agent/nginx-agent.conf

RUN chmod +x /agent/entrypoint.sh
STOPSIGNAL SIGTERM
EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]
