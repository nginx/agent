FROM debian:buster-slim

WORKDIR /mock-management-plane-http
COPY ./build/mock-management-plane-http ./
RUN mkdir -p config/etc/nginx/

CMD ["/mock-management-plane-http/server", "--address", "0.0.0.0:9092", "--configDirectory", "/mock-management-plane-http/config/"]
