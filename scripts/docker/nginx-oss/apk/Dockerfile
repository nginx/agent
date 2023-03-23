ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install-nginx
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

ARG ENTRY_POINT
ARG PACKAGE_NAME

WORKDIR /agent
COPY ./build /agent/build
COPY $ENTRY_POINT /agent/entrypoint.sh

RUN set -x \
    && addgroup -g 101 -S nginx \
    && adduser -S -D -H -u 101 -h /var/cache/nginx -s /sbin/nologin -G nginx -g nginx nginx \
    && apk add \
            ca-certificates \
            wget \
            bash \
            nginx

# Check signing key
# && KEY_SHA512="e7fa8303923d9b95db37a77ad46c68fd4755ff935d0a534d26eba83de193c76166c68bfe7f65471bf8881004ef4aa6df3e34689c305662750c0172fca5d8552a *stdin" \
# && apk add --no-cache --virtual .cert-deps \
#     openssl \
# && wget -O /tmp/nginx_signing.rsa.pub https://nginx.org/keys/nginx_signing.rsa.pub \
# && if [ "$(openssl rsa -pubin -in /tmp/nginx_signing.rsa.pub -text -noout | openssl sha512 -r)" = "$KEY_SHA512" ]; then \
#     echo "key verification succeeded!"; \
#     mv /tmp/nginx_signing.rsa.pub /etc/apk/keys/; \
# else \
#     echo "key verification failed!"; \
#     exit 1; \
# fi \
# && apk del .cert-deps \
# \

RUN chmod +x /agent/entrypoint.sh
STOPSIGNAL SIGTERM

EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]

FROM install-nginx as install-agent

ARG PACKAGE_NAME

# TODO: Validate -y less, Validate --untrust
RUN apk add --allow-untrusted /agent/build/$PACKAGE_NAME.apk
