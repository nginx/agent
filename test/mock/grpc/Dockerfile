FROM debian:buster-slim@sha256:bb3dc79fddbca7e8903248ab916bb775c96ec61014b3d02b4f06043b604726dc

WORKDIR /mock-management-plane-grpc
COPY ./build/mock-management-plane-grpc ./

RUN mkdir config/

CMD ["/mock-management-plane-grpc/server", "--grpcAddress", "0.0.0.0:9092", "--apiAddress", "0.0.0.0:9093", "--configDirectory", "/mock-management-plane-grpc/config", "--logLevel", "DEBUG"]
