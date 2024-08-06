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
COPY ./test/integration/api/nginx.conf /etc/nginx/nginx.conf
COPY ./ /agent

RUN nginx -t
RUN systemctl status nginx

RUN  apt-get update \ 
    && apt install --no-install-recommends --no-install-suggests -y /agent/build/${PACKAGE_NAME}.deb \ 
    &&  rm /agent/build/${PACKAGE_NAME}.deb

RUN unlink /var/log/nginx/access.log
RUN unlink /var/log/nginx/error.log

RUN cat /etc/nginx-agent/nginx-agent.conf
RUN cat /etc/nginx/nginx.conf
