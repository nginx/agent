ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install-nginx
LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

ARG DEBIAN_FRONTEND=noninteractive
ARG ENTRY_POINT
ARG PACKAGE_NAME
ARG PACKAGES_REPO

COPY ./scripts/docker/supervisor.d/nginx.conf etc/supervisor/conf.d/

WORKDIR /agent
COPY ./build /agent/build

RUN set -x \
    && addgroup --system --gid 101 nginx \
    && adduser --system --disabled-login --ingroup nginx --no-create-home --home /nonexistent --gecos "nginx user" --shell /bin/false --uid 101 nginx \
    && apt-get update \
    && apt-get install --no-install-recommends --no-install-suggests -y ca-certificates \
                                                                        gnupg2 \
                                                                        git \
                                                                        make \
                                                                        curl \
                                                                        lsb-release \
                                                                        procps \
                                                                        nginx \
                                                                        supervisor 

# Setup nginx agent repository
RUN curl https://nginx.org/keys/nginx_signing.key | gpg --dearmor | tee /usr/share/keyrings/nginx-archive-keyring.gpg >/dev/null \
    && printf "deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] http://${PACKAGES_REPO}/nginx-agent/ubuntu/ `lsb_release -cs` agent\n" > /etc/apt/sources.list.d/nginx-agent.list

STOPSIGNAL SIGQUIT

EXPOSE 80 443

ENTRYPOINT ["/usr/bin/supervisord", "-c","/etc/supervisor/supervisord.conf", "-u", "root", "--nodaemon", "--logfile", "/dev/stdout", "--logfile_maxbytes", "0" ]

FROM install-nginx as install-agent-local

ARG PACKAGE_NAME

RUN apt install -y /agent/build/$PACKAGE_NAME.deb

FROM install-nginx as install-agent-repo

RUN apt-get update && apt-get install -y nginx-agent
