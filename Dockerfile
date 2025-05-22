# syntax=docker/dockerfile:1.11
FROM docker.io/nginx/nginx-ingress:plus-agentv3

COPY build/nginx-agent-3.0.0-SNAPSHOT-6e70fd64.deb /agent.deb
USER root
RUN apt-get remove -y nginx-agent
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y -f /agent.deb
USER 101
