ARG DEVOPS_DOCKER_URL
ARG GO_LANG_CONTAINER

FROM ${GO_LANG_CONTAINER} AS builder
ENV GOFLAGS="-mod=vendor"
ADD ./ /go_src 
RUN cd /go_src/test/integration/ && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -v -timeout 5m -c -o integration-test

FROM ${DEVOPS_DOCKER_URL}/dockerhub-remote/ubuntu:22.04
ARG METRICS_MODULE_PKG
ARG NGINX_PLUS_VERSION
ADD ./ /go_src 

COPY ${METRICS_MODULE_PKG} /module.tar.gz

RUN mkdir -p /etc/ssl/nginx \
  && mv /go_src/build/nginx-repo.crt /etc/ssl/nginx/nginx-repo.crt && mv /go_src/build/nginx-repo.key /etc/ssl/nginx/nginx-repo.key \
  && chmod +r /etc/ssl/nginx/*.* \
  && set -ex \
  && apt-get update \
  && apt-get install --no-install-recommends --no-install-suggests -y \
    curl \
    gnupg \
    ca-certificates \
    apt-transport-https \
    lsb-release \
    procps \
  && \
  NGINX_GPGKEY=573BFD6B3D8FBC641079A6ABABF5BD827BD9BF62; \
  for server in \
    hkp://keyserver.ubuntu.com:80 \
    pgp.mit.edu; do \
      echo "Fetching GPG key $NGINX_GPGKEY from $server"; \
      gpg --keyserver "$server" \
          --recv-keys "$NGINX_GPGKEY" \
          && break; \
  done \
  && export APT_KEY_DONT_WARN_ON_DANGEROUS_USAGE=DontWarn \
  && curl -sS https://cs.nginx.com/static/keys/nginx_signing.key | apt-key add - \
  && curl -sS --output /etc/apt/apt.conf.d/90pkgs-nginx https://cs.nginx.com/static/files/90pkgs-nginx \
  && gpg --export "$NGINX_GPGKEY" > /etc/apt/trusted.gpg.d/nginx.gpg \
  && printf "deb https://pkgs.nginx.com/plus/$(lsb_release -is | tr '[:upper:]' '[:lower:]') $(lsb_release -cs) nginx-plus\n" > /etc/apt/sources.list.d/nginx-plus.list \
  && mkdir -p /var/tmp/packages-repository/metrics \
  && tar xf /go_src/build/test/metrics-module.tar.gz -C /var/tmp/packages-repository/metrics/ \
  && apt-key add /var/tmp/packages-repository/metrics/nginx-signing.key \
  && echo "deb file:/var/tmp/packages-repository/metrics/ubuntu $(lsb_release -cs) controller" > /etc/apt/sources.list.d/nginx-metrics.list \
  && apt-get update \
  && apt-get install -y -q nginx-plus=${NGINX_PLUS_VERSION}~$(lsb_release -cs) nginx-plus-module-metrics

COPY --from=builder /go_src/test/integration/integration-test /integration-test

ENTRYPOINT ["/integration-test"]
