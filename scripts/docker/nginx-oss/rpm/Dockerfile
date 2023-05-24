ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install-nginx
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

ARG NGINX_CONF
ARG ENTRY_POINT
ARG OS_VERSION
ARG OS_RELEASE

WORKDIR /agent
COPY ./ /agent
COPY $ENTRY_POINT /agent/entrypoint.sh

RUN if [ "$OS_VERSION" = "7" ] && [ "$OS_RELEASE" = "oraclelinux" ]; \
        then yum install -y oracle-epel-release-el7; \
    fi

RUN if [ "$OS_RELEASE" = "centos" ] && [ "$OS_VERSION" = "7" ]; \
        then yum install -y epel-release; \
    fi

RUN if [ "$OS_RELEASE" = "amazonlinux" ]; \
        then amazon-linux-extras enable epel && yum clean metadata \
        && yum install -y epel-release shadow-utils; \
    fi

RUN set -x \
    && groupadd --system --gid 101 nginx \
    && adduser -g nginx --system --no-create-home --home /nonexistent --shell /bin/false --uid 101 nginx \
    && usermod -s /sbin/nologin nginx \
    && usermod -L nginx \
    && yum install -y git \
                      wget \
                      procps \
                      make \
                      nginx

RUN chmod +x /agent/entrypoint.sh
# COPY ./test/integration/api/nginx.conf /etc/nginx/nginx.conf

STOPSIGNAL SIGTERM

EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]


FROM install-nginx as install-agent-local

ARG PACKAGE_NAME

RUN yum localinstall -y /agent/build/$PACKAGE_NAME.rpm

FROM install-nginx as install-agent-repo

ARG PACKAGES_REPO

RUN printf "[nginx-agent] \n\
name=nginx-agent repo \n\
baseurl=https://$PACKAGES_REPO/nginx-agent/centos/$OS_VERSION/\$basearch \n\
sslclientcert=/etc/ssl/nginx/nginx-repo.crt \n\
sslclientkey=/etc/ssl/nginx/nginx-repo.key \n\
gpgcheck=1 \n\
enabled=1 \n\
sslverify=true \n\
gpgkey=https://nginx.org/keys/nginx_signing.key" > /etc/yum.repos.d/nginx-agent.repo  \
    && yum install -y nginx-agent \
    && yum clean all \
    && rm -rf /var/cache/yum /etc/yum.repos.d/* /etc/ssl/nginx
