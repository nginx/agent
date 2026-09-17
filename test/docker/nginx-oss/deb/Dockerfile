ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install-nginx
LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

ARG DEBIAN_FRONTEND=noninteractive
ARG ENTRY_POINT
ARG PACKAGE_NAME
ARG PACKAGES_REPO
ARG OS_RELEASE

WORKDIR /agent
COPY ./build /agent/build
COPY $ENTRY_POINT /agent/entrypoint.sh

# Pin bullseye apt to the snapshot the debian/eol image was built from so
# preinstalled packages match repo indices (avoiding archive.debian.org's
# pruned bullseye-security pool and drift against the image's baked-in state).
# These are the URLs the debian/eol maintainer ships pre-configured (commented)
# in the image's own /etc/apt/sources.list.
RUN if grep -q '^VERSION_CODENAME=bullseye' /etc/os-release; then \
        SNAPSHOT=20250809T133719Z; \
        { \
            echo "deb http://snapshot.debian.org/archive/debian-archive/$SNAPSHOT/debian bullseye main"; \
            echo "deb http://snapshot.debian.org/archive/debian-security/$SNAPSHOT bullseye-security main"; \
            echo "deb http://snapshot.debian.org/archive/debian-archive/$SNAPSHOT/debian bullseye-updates main"; \
        } > /etc/apt/sources.list; \
        echo 'Acquire::Check-Valid-Until "false";' > /etc/apt/apt.conf.d/99no-check-valid-until; \
    fi

RUN set -x \
    && groupadd --system --gid 101 nginx \
    && useradd --system --gid nginx --no-create-home --home-dir /nonexistent --uid 101 nginx \
    && apt-get update \
    && apt-get install --no-install-recommends --no-install-suggests -y ca-certificates \
                                                                        gnupg2 \
                                                                        git \
                                                                        make \
                                                                        curl \
                                                                        lsb-release \
                                                                        procps \
                                                                        nginx
# Setup nginx agent repository
RUN curl https://nginx.org/keys/nginx_signing.key | gpg --dearmor | tee /usr/share/keyrings/nginx-archive-keyring.gpg >/dev/null \
    && printf "deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] http://${PACKAGES_REPO}/nginx-agent/${OS_RELEASE}/ `lsb_release -cs` agent\n" > /etc/apt/sources.list.d/nginx-agent.list

RUN chmod +x /agent/entrypoint.sh
STOPSIGNAL SIGTERM

EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]

FROM install-nginx as install-agent-local

ARG PACKAGE_NAME

RUN apt install -y /agent/build/$PACKAGE_NAME.deb

FROM install-nginx as install-agent-repo

ARG NGINX_AGENT_VERSION

RUN apt-get update
RUN if [ -z "$NGINX_AGENT_VERSION" ]; then apt-get install -y nginx-agent; else apt-get install -y nginx-agent=${NGINX_AGENT_VERSION}; fi
