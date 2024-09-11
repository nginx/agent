# NGINX Receiver

This receiver can fetch stats from a NGINX instance by using two different ways of scraping:  
* Scarping metrics from the `ngx_http_stub_status_module` module's `status` endpoint.
* Scraping metrics from the NGINX access logs.

## Configuration

### NGINX Module

You must configure NGINX to expose status information by editing the NGINX
configuration.  Please see
[ngx_http_stub_status_module](http://nginx.org/en/docs/http/ngx_http_stub_status_module.html)
for a guide to configuring the NGINX stats module `ngx_http_stub_status_module`.

You must also configure an access log as well. Please see [Setting Up the Access Log](https://docs.nginx.com/nginx/admin-guide/monitoring/logging/#setting-up-the-access-log) for a guide to configuring an access log.

### Receiver Config

The following settings are required:

- `endpoint` (default: `http://localhost:80/status`): The URL of the NGINX status endpoint

The following settings are optional:

- `collection_interval` (default = `10s`): This receiver collects metrics on an interval. This value must be a string readable by Golang's [time.ParseDuration](https://pkg.go.dev/time#ParseDuration). Valid time units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`.

- `initial_delay` (default = `1s`): defines how long this receiver waits before starting.

- `access_logs` (default = `[]`): defines a list of access logs to scrape.
    - `file_path`: The file path to the access log.
    - `log_format`: The format of the access log.

Example:

```yaml
receivers:
  nginx:
    endpoint: "http://localhost:80/status"
    collection_interval: 10s
    access_logs:
      - log_format: "$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\" \"$http_x_forwarded_for\"\"$upstream_cache_status\""
        file_path: "/var/log/nginx/access-custom.conf"
```
