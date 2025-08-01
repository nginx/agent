FROM golang:bookworm@sha256:ef8c5c733079ac219c77edab604c425d748c740d8699530ea6aced9de79aea40

WORKDIR /mock-management-otel-collector
COPY ./build/mock-management-otel-collector ./

CMD ["/mock-management-otel-collector/collector"]
