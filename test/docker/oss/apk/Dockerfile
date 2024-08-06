ARG CONTAINER_NGINX_IMAGE_REGISTRY
ARG TAG
ARG IMAGE_PATH
FROM ${CONTAINER_NGINX_IMAGE_REGISTRY}${IMAGE_PATH}:alpine AS install
LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

ARG OS_RELEASE
ARG OS_VERSION
ARG PACKAGE_NAME
ARG CONTAINER_OS_TYPE

WORKDIR /agent
COPY ./build/${PACKAGE_NAME}.${CONTAINER_OS_TYPE} /agent/build/${PACKAGE_NAME}.${CONTAINER_OS_TYPE}
COPY ./ /agent

RUN apk add --allow-untrusted /agent/build/${PACKAGE_NAME}.${CONTAINER_OS_TYPE}

RUN unlink /var/log/nginx/access.log
RUN unlink /var/log/nginx/error.log
