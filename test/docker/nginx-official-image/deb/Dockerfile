ARG CONTAINER_NGINX_IMAGE_REGISTRY
ARG TAG
ARG IMAGE_PATH
FROM ${CONTAINER_NGINX_IMAGE_REGISTRY}${IMAGE_PATH}:${TAG} AS install
LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

ARG OS_RELEASE
ARG OS_VERSION
ARG DEBIAN_FRONTEND=noninteractive
ARG PACKAGE_NAME
ARG PACKAGES_REPO
ARG CONTAINER_OS_TYPE
ARG ARCH

WORKDIR /agent
COPY ./build/${PACKAGE_NAME}.${CONTAINER_OS_TYPE} /agent/build/${PACKAGE_NAME}.${CONTAINER_OS_TYPE}

RUN  apt-get update \
    && apt install --no-install-recommends --no-install-suggests --allow-downgrades -y /agent/build/${PACKAGE_NAME}.${CONTAINER_OS_TYPE} \
    &&  rm /agent/build/${PACKAGE_NAME}.${CONTAINER_OS_TYPE}

RUN if [ "$TAG" == "mainline-bookworm" || "$TAG" == "stable-bookworm" ]; then \
     curl -O https://${PACKAGES_REPO}/nginx-agent/${OS_RELEASE}/pool/agent/n/nginx-agent/nginx-agent_3.2.1~bookworm_${ARCH}.deb fi

RUN unlink /var/log/nginx/access.log
RUN unlink /var/log/nginx/error.log
