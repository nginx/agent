# NGINX Plus Receiver

This receiver can fetch stats from a NGINX Plus instance via the `ngx_http_api_module` module's `api` endpoint.

## Configuration

### NGINX Module

You must configure NGINX Plus to expose api information by editing the NGINX configuration.  
Please see [ngx_http_api_module](https://nginx.org/en/docs/http/ngx_http_api_module.html) for a guide to configuring the NGINX stats module `ngx_http_api_module`.

### Receiver Config

The following settings are required:


The following settings are required:
- `api_details`: Details for the NGINX Plus API endpoint.
    - `url`: (default: `http://localhost:80/api`): The URL of the NGINX Plus API endpoint.
    - `listen`: (default: `localhost:80`): The listen directive of the NGINX Plus API endpoint.
    - `location`: (default: `/api`): The location directive of the NGINX Plus API endpoint.

The following settings are optional:

- `collection_interval` (default = `10s`): This receiver collects metrics on an interval. This value must be a string readable by Golang's [time.ParseDuration](https://pkg.go.dev/time#ParseDuration). Valid time units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`.

- `initial_delay` (default = `1s`): defines how long this receiver waits before starting.

Example:

```yaml
receivers:
  nginxplus:
    api_details:
      url: "http://localhost:80/api"
      listen: "localhost:80"
      location: "/api"
    collection_interval: 10s
```
