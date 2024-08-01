ARG CONTAINER_NGINX_IMAGE_REGISTRY
ARG TAG
ARG IMAGE_PATH
FROM ${CONTAINER_NGINX_IMAGE_REGISTRY}${IMAGE_PATH}:${TAG} AS install
LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

ARG OS_RELEASE
ARG OS_VERSION
ARG DEBIAN_FRONTEND=noninteractive
ARG PACKAGE_NAME

WORKDIR /agent
COPY ./build/${PACKAGE_NAME}.deb /agent/build/${PACKAGE_NAME}.deb
COPY ./ /agent

FROM install AS runtime
RUN ls /var/log/nginx
RUN unlink /var/log/nginx/access.log
RUN unlink /var/log/nginx/error.log
