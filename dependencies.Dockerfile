# DO NOT BUILD
# This file is just for tracking dependencies of the semantic convention build.
# Dependabot can keep this file up to date with latest containers.

# Weaver is used to generate markdown docs, and enforce policies on the model.
FROM otel/weaver:v0.24.1@sha256:263964a7d444e77812f7a2d654e17683c4760a968c91278acdb7a44c20ccd572 AS weaver

# OPA is used to test policies enforced by weaver.
FROM openpolicyagent/opa:1.17.1@sha256:dc009236137bb225a1ef09293bb32f2ee1861cc428870d297bf71412d50221c3 AS opa

# Semconv gen is used for backwards compatibility checks.
# TODO(jsuereth): Remove this when no longer used.
FROM otel/semconvgen:0.25.0@sha256:9df7b8cbaa732277d64d0c0a8604d96bb6f5a36d0e96338cba5dced720c16485 AS semconvgen
