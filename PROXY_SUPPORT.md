# Agent Proxy Support

This document describes how to configure the NGINX Agent to connect to the management plane through an explicit forward proxy (EFP), via HTTP/1.1 and authentication.

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

- `url`: Proxy URL (http supported)
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
