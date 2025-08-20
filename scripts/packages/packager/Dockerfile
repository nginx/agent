ARG package_type

FROM docker.io/golang@sha256:62ba6b19de03e891f7fa1001326bd48411f2626ff35e7ba5b9d890711ce581d9 AS base

ARG PKG_VER="1.17.5"
ARG PKG_DIR="/tmp/pkg"

RUN apt-get update && \
    apt-get install -y make jq gnupg gnupg1 gpgv1 git aptly debsig-verify createrepo-c dnf rpm \
                       curl gettext-base make monkeysphere libtool unzip libssl-dev libbz2-dev libbsd-dev libarchive-dev liblzma-dev zlib1g-dev

# compile, install pkg tool for linux
RUN	mkdir $PKG_DIR; cd $PKG_DIR; \
	go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.32.0; \
    curl -L -o pkg.zip https://github.com/freebsd/pkg/archive/refs/tags/$PKG_VER.zip; \
	unzip -qo pkg.zip; cd pkg-$PKG_VER; ./configure; make -s -j$(nproc); make install; \
	rm -rf $PKG_DIR

FROM base AS local-package
ADD ./scripts/packages/packager/local-entrypoint.sh /scripts/entrypoint.sh
RUN chmod +x /scripts/entrypoint.sh

FROM base AS signed-package
ADD ./scripts/packages/packager/signed-entrypoint.sh /scripts/entrypoint.sh
RUN chmod +x /scripts/entrypoint.sh

FROM ${package_type} AS final
ENTRYPOINT [ "/scripts/entrypoint.sh" ]
