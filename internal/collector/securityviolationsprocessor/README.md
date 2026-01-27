# Security Violations Processor

OpenTelemetry Collector processor that transforms NGINX App Protect security violation syslog messages into structured protobuf events.

## What It Does

Processes NGINX App Protect WAF syslog messages and transforms them into `SecurityViolationEvent` protobuf messages:

1. Parses RFC3164 syslog messages (best-effort mode)
2. Extracts CSV formatted data from NAP `secops_dashboard` log profile
3. Parses XML violation details with context extraction (parameter, header, cookie, uri, request)
4. Extracts attack signature details
5. Outputs structured protobuf events for downstream consumption

## Implementation

| File | Purpose |
|------|---------|
| [`processor.go`](processor.go) | Main processor implementation, RFC3164 parsing, orchestration |
| [`csv_parser.go`](csv_parser.go) | CSV parsing and field mapping |
| [`violations_parser.go`](violations_parser.go) | XML parsing, context extraction, signature parsing |
| [`xml_structs.go`](xml_structs.go) | XML structure definitions (BADMSG, violation contexts) |
| [`helpers.go`](helpers.go) | Utility functions |

See individual files for implementation details. Protobuf schema defined in [`api/grpc/events/v1/security_violation.proto`](../../../api/grpc/events/v1/security_violation.proto).

## Requirements

- **Input**: NAP syslog messages with `secops_dashboard` log profile (33 CSV fields)
- **Output**: `SecurityViolationEvent` protobuf messages

## Testing

```bash
# Run all tests
go test ./internal/collector/securityviolationsprocessor -v

# Check coverage
go test ./internal/collector/securityviolationsprocessor -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Test coverage: CSV parsing, XML parsing (5 violation contexts), encoding edge cases, error handling.

## Error Handling

Implements graceful degradation:
- Malformed XML: Logs warning, continues processing
- Base64 decode errors: Falls back to raw data
- Missing fields: Uses empty strings
- Context inference: Derives from violation names when not explicit
