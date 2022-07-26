FROM ubuntu:22.04 as install
LABEL maintainer="NGINX Agent Maintainers <agent@nginx.com>"

WORKDIR /agent
COPY ./build/nginx-agent.deb /agent/nginx-agent.deb
COPY ./entrypoint.sh /agent/entrypoint.sh
COPY ./nginx-agent.conf /agent/nginx-agent.conf
COPY ./nginx.conf /agent/nginx.conf

RUN set -x \
    && addgroup --system --gid 101 nginx \
    && adduser --system --disabled-login --ingroup nginx --no-create-home --home /nonexistent --gecos "nginx user" --shell /bin/false --uid 101 nginx \
    && apt-get update \
    && apt-get install --no-install-recommends --no-install-suggests -y \
                        ca-certificates \
                        gnupg1 \
                        lsb-release \
                        git \
                        wget \
                        make \
                        curl \
                        vim \
    && apt-get update \
    && apt-get install -y build-essential libpcre3 libpcre3-dev zlib1g zlib1g-dev libssl-dev \
    && wget http://nginx.org/download/nginx-1.23.2.tar.gz \
    && tar xfz nginx-1.23.2.tar.gz \
    && cd nginx-1.23.2 \
    && cp /agent/nginx.conf ./conf/nginx.conf \
    && ./configure --with-http_stub_status_module \
    && make \
    && make install \
    && apt-get install -y -f /agent/nginx-agent.deb 

# run the nginx and agent
FROM install as runtime

COPY --from=install /agent/entrypoint.sh /agent/entrypoint.sh
COPY --from=install /agent/nginx-agent.conf /etc/nginx-agent/nginx-agent.conf

RUN chmod +x /agent/entrypoint.sh
STOPSIGNAL SIGTERM
EXPOSE 8080

ENTRYPOINT ["/agent/entrypoint.sh"]
