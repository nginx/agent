ARG CONTAINER_NGINX_IMAGE_REGISTRY
ARG TAG
FROM ${CONTAINER_NGINX_IMAGE_REGISTRY}:${TAG} AS install-agent-local
LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

ARG OS_RELEASE
ARG OS_VERSION
ARG DEBIAN_FRONTEND=noninteractive
ARG PACKAGE_NAME

WORKDIR /agent
COPY ./build/${PACKAGE_NAME} /agent/build/${PACKAGE_NAME}
COPY ./ /agent

RUN  apt-get update \ 
    && apt install --no-install-recommends --no-install-suggests -y /agent/build/${PACKAGE_NAME} \ 
    &&  rm /agent/build/${PACKAGE_NAME}

FROM install AS runtime

COPY --from=install /agent /agent
