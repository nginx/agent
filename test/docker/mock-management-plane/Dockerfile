FROM debian:buster-slim

WORKDIR /mock-management-plane
COPY ./build/mock-management-plane ./
RUN mkdir config

CMD ["/mock-management-plane/server", "--address", "0.0.0.0:9092", "--configDirectory", "/mock-management-plane/config/"]
