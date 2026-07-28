# DO NOT BUILD
# This file is just for tracking dependencies of the semantic convention build.
# Dependabot can keep this file up to date with latest containers.

# Weaver is used to generate markdown docs, and enforce policies on the model.
FROM otel/weaver:v0.25.0@sha256:bef6000b4a4be46f81242f9ee785e0ebf0604606c15f92cb54a59893a741ec0c AS weaver

# OPA is used to test policies enforced by weaver.
FROM openpolicyagent/opa:1.18.2@sha256:cba27d3c6af2feba1e4d6e6b5e24df5b53db332420d4148a90acccd12efae6ed AS opa

# Semconv gen is used for backwards compatibility checks.
# TODO(jsuereth): Remove this when no longer used.
FROM otel/semconvgen:0.25.0@sha256:9df7b8cbaa732277d64d0c0a8604d96bb6f5a36d0e96338cba5dced720c16485 AS semconvgen
