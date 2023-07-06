ARG BASE_IMAGE
FROM ${BASE_IMAGE} as install-nginx
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

ARG PACKAGES_REPO
ARG ENTRY_POINT
ARG OS_VERSION
ARG OS_RELEASE

WORKDIR /agent
COPY ./ /agent
COPY $ENTRY_POINT /agent/entrypoint.sh

RUN if [ "$OS_VERSION" = "7" ] && [ "$OS_RELEASE" = "oraclelinux" ]; \
        then yum install -y oracle-epel-release-el7; \
    fi

RUN if [ "$OS_VERSION" = "2" ] && [ "$OS_RELEASE" = "amazonlinux" ]; \
        then amazon-linux-extras enable epel && yum clean metadata \
        && yum install -y epel-release; \
    fi

RUN if [ "$OS_RELEASE" = "amazonlinux" ]; \
        then yum install -y shadow-utils; \
    fi

RUN if [ "$OS_RELEASE" = "centos" ] && [ "$OS_VERSION" = "7" ]; \
        then yum install -y epel-release; \
    fi

RUN if [ "$OS_RELEASE" = "redhatenterprise" ] && [ "$OS_VERSION" != "9" ]; \
        then printf "[nginx] \n\
name=nginx repo \n\
baseurl=http://nginx.org/packages/mainline/rhel/${OS_VERSION}/\$basearch/ \n\
gpgcheck=0 \n\
enabled=1" > /etc/yum.repos.d/nginx.repo; \
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

# Setup nginx agent repository
RUN if [ "$OS_VERSION" = "2023" ] && [ "$OS_RELEASE" = "amazonlinux" ]; \
    then printf "[nginx-agent] \n\
name=nginx-agent repo \n\
baseurl=http://${PACKAGES_REPO}/nginx-agent/amzn/${OS_VERSION}/\$basearch \n\
gpgcheck=1 \n\
enabled=1 \n\
gpgkey=https://nginx.org/keys/nginx_signing.key \n\
module_hotfixes=true" > /etc/yum.repos.d/nginx-agent.repo; \
    elif [ "$OS_VERSION" = "2" ] && [ "$OS_RELEASE" = "amazonlinux" ]; \
    then printf "[nginx-agent] \n\
name=nginx-agent repo \n\
baseurl=http://${PACKAGES_REPO}/nginx-agent/amzn2/${OS_VERSION}/\$basearch \n\
gpgcheck=1 \n\
enabled=1 \n\
gpgkey=https://nginx.org/keys/nginx_signing.key \n\
module_hotfixes=true" > /etc/yum.repos.d/nginx-agent.repo; \
    else printf "[nginx-agent] \n\
name=nginx-agent repo \n\
baseurl=http://${PACKAGES_REPO}/nginx-agent/centos/${OS_VERSION}/\$basearch \n\
gpgcheck=1 \n\
enabled=1 \n\
gpgkey=https://nginx.org/keys/nginx_signing.key \n\
module_hotfixes=true" > /etc/yum.repos.d/nginx-agent.repo; \
    fi

RUN chmod +x /agent/entrypoint.sh

STOPSIGNAL SIGTERM

EXPOSE 80 443

ENTRYPOINT ["/agent/entrypoint.sh"]


FROM install-nginx as install-agent-local

ARG PACKAGE_NAME

RUN yum localinstall -y /agent/build/${PACKAGE_NAME}.rpm


FROM install-nginx as install-agent-repo

RUN yum install -y nginx-agent \
    && yum clean all \
    && rm -rf /var/cache/yum /etc/yum.repos.d/* /etc/ssl/nginx
