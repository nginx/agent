# DO NOT BUILD
# This file is just for tracking dependencies of the semantic convention build.
# Dependabot can keep this file up to date with latest containers.

# Weaver is used to generate markdown docs, and enforce policies on the model.
FROM otel/weaver:v0.27.0@sha256:3049b4079049d4abb1b5632f511ada2c33505a1c60f3f8535e93f87f0696f056 AS weaver

# OPA is used to test policies enforced by weaver.
FROM openpolicyagent/opa:1.21.4@sha256:12420358d3f22a0cd2424ddc44a16e292ecb42825d3dfbca779986006822a3f6 AS opa

# Semconv gen is used for backwards compatibility checks.
# TODO(jsuereth): Remove this when no longer used.
FROM otel/semconvgen:0.25.0@sha256:9df7b8cbaa732277d64d0c0a8604d96bb6f5a36d0e96338cba5dced720c16485 AS semconvgen
