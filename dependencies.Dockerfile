# DO NOT BUILD
# This file is just for tracking dependencies of the semantic convention build.
# Dependabot can keep this file up to date with latest containers.

# Weaver is used to generate markdown docs, and enforce policies on the model.
FROM otel/weaver:v0.28.0@sha256:5857bb81d8cc7ed608cdcae37c4b459eb020f21dd4acade35f7933cd6762fbcc AS weaver

# OPA is used to test policies enforced by weaver.
FROM openpolicyagent/opa:1.22.4@sha256:ff50f50332ff9e722ffafe2658c195ced1f9b6d4f3064000c2ecfa8cdf7acccf AS opa

# Semconv gen is used for backwards compatibility checks.
# TODO(jsuereth): Remove this when no longer used.
FROM otel/semconvgen:0.25.0@sha256:9df7b8cbaa732277d64d0c0a8604d96bb6f5a36d0e96338cba5dced720c16485 AS semconvgen
