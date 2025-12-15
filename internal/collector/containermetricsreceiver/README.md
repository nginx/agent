# Container Metrics Receiver

The Container Metrics receiver generates metrics about the container scraped from the cgroup files.

## Configuration

### Receiver Config

The following settings are optional:
- `collection_interval` (default = `10s`): This receiver collects metrics on an interval. This value must be a string readable by Golang's [time.ParseDuration](https://pkg.go.dev/time#ParseDuration). Valid time units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`.
- `initial_delay` (default = `1s`): defines how long this receiver waits before starting.

Example:
```yaml
containermetrics:
  collection_interval: <duration> # default = 1m
  initial_delay: <duration> # default = 1s
```

The available scrapers are:

| Scraper      | Supported OSs                | Description                                            |
| ------------ | ---------------------------- | ------------------------------------------------------ |
| [cpu]        | All                          | CPU utilization metrics                                |
| [memory]     | All                          | Memory utilization metrics                             |

[cpu]: ./internal/scraper/cpuscraper/documentation.md
[memory]: ./internal/scraper/memoryscraper/documentation.md