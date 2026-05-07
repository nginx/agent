# DO NOT BUILD
# This file is just for tracking dependencies of the semantic convention build.
# Dependabot can keep this file up to date with latest containers.

# Weaver is used to generate markdown docs, and enforce policies on the model.
FROM otel/weaver:v0.23.0@sha256:7984ecb55b859eb3034ae9d836c4eeda137e2bdd0873b7ba2bb6c3d24d6ff457 AS weaver

# OPA is used to test policies enforced by weaver.
FROM openpolicyagent/opa:1.15.2@sha256:4750afa83e85a1b0a4964ea5700e9caa1c1e61b508435e09fd77f038747340e6 AS opa

# Semconv gen is used for backwards compatibility checks.
# TODO(jsuereth): Remove this when no longer used.
FROM otel/semconvgen:0.25.0@sha256:9df7b8cbaa732277d64d0c0a8604d96bb6f5a36d0e96338cba5dced720c16485 AS semconvgen
