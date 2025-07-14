FROM debian:buster-slim

WORKDIR /mock-management-plane-grpc
COPY ./build/mock-management-plane-grpc ./

RUN mkdir config/

CMD ["/mock-management-plane-grpc/server", "--grpcAddress", "0.0.0.0:9095", "--apiAddress", "0.0.0.0:9096", "--configDirectory", "/mock-management-plane-grpc/config", "--logLevel", "DEBUG"]
