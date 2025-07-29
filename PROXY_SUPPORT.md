# Agent Proxy Support

This document describes how to configure the NGINX Agent to connect to the management plane through an explicit forward proxy (EFP), including HTTP/1.1 and HTTPS proxies, authentication, and TLS options. It also covers proxying for the embedded OpenTelemetry (OTel) Collector.

---

## 1. Basic Proxy Configuration

Add a `proxy` section under the `server` block in your agent config file:

```yaml
server:
  host: mgmt.example.com
  port: 443
  type: 1
  proxy:
    url: "http://proxy.example.com:3128"
    timeout: 10s
```

- `url`: Proxy URL (http or https supported)
- `timeout`: Dial timeout for connecting to the proxy

---

## 2. Proxy Authentication

### Basic Auth
```yaml
proxy:
  url: "http://proxy.example.com:3128"
  auth_method: "basic"
  username: "user"
  password: "pass"
```

### Bearer Token
```yaml
proxy:
  url: "http://proxy.example.com:3128"
  auth_method: "bearer"
  token: "your_token_here"
```

---

## 3. HTTPS Proxy (TLS)

If your proxy requires TLS (https), use:
```yaml
proxy:
  url: "https://proxy.example.com:3129"
  timeout: 10s
  tls:
    ca: "/path/to/proxy-ca.pem"           # Trusted CA for proxy
    cert: "/path/to/client-cert.pem"      # Optional client certificate
    key: "/path/to/client-key.pem"        # Optional client key
    server_name: "proxy.example.com"       # SNI/hostname for proxy
    skip_verify: false                     # (true to skip proxy cert validation)
```

---

## 4. Example Full Agent Config

```yaml
server:
  host: mgmt.example.com
  port: 443
  type: 1
  proxy:
    url: "https://proxy.example.com:3129"
    auth_method: "bearer"
    token: "your_token_here"
    timeout: 10s
    tls:
      ca: "/path/to/proxy-ca.pem"
      cert: "/path/to/client-cert.pem"
      key: "/path/to/client-key.pem"
      server_name: "proxy.example.com"
      skip_verify: false
```

---

## 5. OTel Exporter Proxy Support

To proxy telemetry exported by the embedded OTel Collector, add a `proxy` section under the `exporters` block:

```yaml
exporters:
  otlp:
    endpoint: <your-endpoint>
    proxy:
      http: http://proxy.example.com:3128
      https: http://proxy.example.com:3128
```

- `no_proxy`: Comma-separated list of hosts/domains to bypass the proxy (optional, applies to both agent and OTel Collector traffic).

The agent sets `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment variables before starting the embedded OTel Collector. All OTel exporters will use these for outbound HTTP(S) connections.

---

## 6. Notes
- All proxy features work with both HTTP and HTTPS management plane endpoints.
- The agent tunnels all gRPC (HTTP/2) traffic through the proxy using HTTP/1.1 CONNECT.
- For Squid or similar, check logs for `CONNECT` lines to verify traffic.
- All proxy errors are logged; check agent logs for troubleshooting.
- If telemetry is not reaching your backend, ensure the proxy allows outbound connections to the required endpoints.
- Use the `no_proxy` field to bypass the proxy for internal or localhost addresses.
- Check your proxy server logs to verify that agent and OTel Collector traffic is routed as expected.
- You can also block direct egress to confirm that only proxied traffic succeeds.

---

## 7. Security Note

If your proxy requires authentication, include the credentials in the URL (e.g., `http://user:pass@host:port`).  
**Ensure your config files are secured, as proxy credentials may be sensitive.**

---

## 8. Environment Variable Override (Advanced)

You may also manually set `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment variables before starting the agent. These will override any values set via the YAML config.

---

## 9. FAQ

**Q: What happens if both config and environment variables are set?**  
A: Manually set environment variables will take precedence over config values.

**Q: How do I debug proxy issues?**  
A: Check agent and proxy logs for connection attempts and failures. Use `no_proxy` to exclude local addresses if needed.

---
