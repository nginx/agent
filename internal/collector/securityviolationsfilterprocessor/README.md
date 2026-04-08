# Security violations filter processor

Internal processor in the NGINX Agent collector pipeline for NGINX App Protect security violation logs.

This processor does not parse or transform the log body. Its job is to verify that the incoming log body appears to match the expected `secops-dashboard-log` pipe-delimited format before forwarding records downstream.

## What it does

For log records:

- Validates the first string log body seen by the processor.
- Expects a pipe-delimited body with exactly 28 fields.
- If validation succeeds, forwards records unchanged.
- Adds resource attributes:
  - `csv.schema.name=secops-dashboard-log`
  - `csv.schema.version=1.0`

For non-log signals:

- This processor only implements log processing behavior.

## Expected field order

The expected `|`-delimited field order is:

```text
%support_id%|%ip_client%|%src_port%|%dest_ip%|%dest_port%|%vs_name%|%policy_name%|%method%|%uri%|%protocol%|%request_status%|%response_code%|%outcome%|%outcome_reason%|%violation_rating%|%blocking_exception_reason%|%is_truncated_bool%|%sig_ids%|%sig_names%|%sig_cves%|%sig_set_names%|%threat_campaign_names%|%sub_violations%|%x_forwarded_for_header_value%|%violations%|%violation_details%|%request%|%geo_location%
```

`%geo_location%` must be present as the final field.

## Gate behavior

This processor uses a one-time gate:

- Initial state: pending.
- First valid string body opens the gate.
- First invalid body closes the gate.

Once the gate is closed, all subsequent security violation log records are dropped until the OpenTelemetry collector is restarted.

The gate closes when the first inspected record is:

- Not a string body.
- A string body with a field count other than 28.

This is intentional. It prevents mixed or unexpected logging formats from being forwarded as if they matched the expected schema.

## Important operational notes

- Validation is based on field count only. The processor does not inspect individual field names or parse field contents.
- Because the decision is made on the first inspected record, startup ordering matters. If the first security violation record uses the wrong format, later valid records will still be dropped until restart.
- This processor passes the original body through unchanged. Parsing and field extraction happen in downstream components.
- If logs are unexpectedly missing, verify the NGINX App Protect logging profile is using the exact `secops-dashboard-log` field order shown above.

## Configuration

No processor-specific configuration is required.
