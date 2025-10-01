{% extends "/layouts/doc.html" %}
{% macro Title string %}Monitoring{% end %}
{% Article %}

# Monitoring Meergo

Meergo monitors its own internal service metrics, and makes them available under the `/metrics` path.

## Metrics

 Name                                         | Type         | Labels                       | Description                                                                 |
|---------------------------------------------|--------------|------------------------------|-----------------------------------------------------------------------------|
| `meergo_db_acquired_conns`                  | Gauge        | –                            | Current number of connections in use by the database connection pool.       |
| `meergo_db_max_conns`                       | Gauge        | –                            | Configured maximum number of simultaneous database connections.             |
| `meergo_db_acquire_duration_seconds_total`  | Counter      | –                            | Cumulative seconds spent acquiring connections from the database pool.      |
| `meergo_db_acquire_count_total`             | Counter      | –                            | Total number of successful database connection acquisitions.                |
| `meergo_db_new_conns_count_total`           | Counter      | –                            | Total number of newly created database connections (connection churn).      |
| `meergo_lambda_errors_total`                | CounterVec   | `type`                       | Total number of Lambda errors, categorized by error type.                   |
| `meergo_lambda_duration_seconds`            | Histogram    | –                            | Duration of successful Lambda executions, in seconds.                       |
| `meergo_lambda_records_total`               | Counter      | –                            | Total number of input records processed by successful Lambda executions.    |
| `meergo_sender_queue_available`             | GaugeVec     | –                            | Number of available events in the event queue.                              |
| `meergo_sender_queue_wait`                  | HistogramVec | `connector`,<br>`connection` | Time spent waiting in the event queue (in seconds).                         |

### Notes

- The total number of successful Lambda executions is provided by the metric `meergo_lambda_duration_seconds_count`, which is part of the` meergo_lambda_duration_seconds` histogram. It represents the total number of observations (i.e., completed executions) recorded in the histogram.
- Possible values for the `type` label in `meergo_lambda_errors_total`:  
  `network`, `lambda_internal`, `function_not_found`, `serialization`, and `function_exec`.
- Buckets defined for `meergo_lambda_duration_seconds`: `0.1`, `0.5`, `1`, `2.5`,  and `5` (in seconds)
- Buckets defined for `meergo_sender_queue_wait`: `0.005`, `0.01`, `0.025`, `0.05`, `0.075`, `0.1`, `0.15`, `0.2`, `0.3`, `0.5`, `0.75`, `1.0`, `2.0` (in seconds)

## Prometheus

To monitor these metrics with [Prometheus](https://prometheus.io/), configure it with the following scrape job:

```yaml
scrape_configs:
  - job_name: 'meergo'
    static_configs:
      - targets: [ "127.0.0.1:2022" ]
```

## Grafana

After setting up Prometheus to collect the metrics, configure [Grafana](https://grafana.com/oss/grafana/) to display them in dashboards.
