ARG CONTAINER_NGINX_IMAGE_REGISTRY
ARG TAG
FROM ${CONTAINER_NGINX_IMAGE_REGISTRY}:${TAG} AS install
LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

ARG DEBIAN_FRONTEND=noninteractive

WORKDIR /agent
COPY ./build/${PACKAGE_NAME}.deb /agent/build/${PACKAGE_NAME}.deb
COPY ./ /agent

FROM install AS runtime
RUN ls /var/log/nginx
RUN unlink /var/log/nginx/access.log
RUN unlink /var/log/nginx/error.log
