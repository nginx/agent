[comment]: <> (Code generated by mdatagen. DO NOT EDIT.)

# nginx

## Default Metrics

The following metrics are emitted by default. Each of them can be disabled by applying the following configuration:

```yaml
metrics:
  <metric_name>:
    enabled: false
```

### nginx.connections_accepted

The total number of accepted client connections

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| connections | Sum | Int | Cumulative | true |

### nginx.connections_current

The current number of nginx connections by state

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| connections | Sum | Int | Cumulative | false |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| state | The state of a connection | Str: ``active``, ``reading``, ``writing``, ``waiting`` |

### nginx.connections_handled

The total number of handled connections. Generally, the parameter value is the same as nginx.connections_accepted unless some resource limits have been reached (for example, the worker_connections limit).

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| connections | Sum | Int | Cumulative | true |

### nginx.http.response.status

The number of responses, grouped by status code range.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| responses | Sum | Int | Cumulative | true |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| nginx.status_range | A status code range or bucket for a HTTP response's status code. | Str: ``1xx``, ``2xx``, ``3xx``, ``4xx``, ``5xx`` |

### nginx.requests

Total number of requests made to the server since it started

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| requests | Sum | Int | Cumulative | true |