# Syslog Processor

Internal component of the NGINX Agent that processes syslog messages. Parses RFC3164 formatted syslog entries from log records and extracts structured attributes. Successfully parsed messages have their body replaced with the clean message content.

Part of the NGINX Agent's log collection pipeline.