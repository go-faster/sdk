# sdk [![Go Reference](https://img.shields.io/badge/go-pkg-00ADD8)](https://pkg.go.dev/github.com/go-faster/sdk#section-documentation) [![codecov](https://img.shields.io/codecov/c/github/go-faster/sdk?label=cover)](https://codecov.io/gh/go-faster/sdk) [![alpha](https://img.shields.io/badge/-alpha-orange)](https://go-faster.org/docs/projects/status#alpha)

SDK for go-faster applications.
Implements automatic setup of observability and daemonization based on environment variables.

## Packages

| Package      | Description                                                |
|--------------|------------------------------------------------------------|
| `autometer`  | Automatic OpenTelemetry MeterProvider from environment     |
| `autotracer` | Automatic OpenTelemetry TracerProvider from environment    |
| `autologs`   | Automatic OpenTelemetry LoggerProvider from environment    |
| `autopyro`   | Automatic Grafana Pyroscope configuration from environment |
| `profiler`   | Explicit pprof routes                                      |
| `zctx`       | context.Context and tracing support for zap                |
| `gold`       | Golden files in tests                                      |
| `app`        | Automatic setup observability and run daemon               |
| `autometric` | Reflect-based OpenTelemetry metric initializer             |
| `otelsync`   | OpenTelemetry synchronous adapter for async metrics        |

## Environment variables

> [!WARNING]
> The pprof listener is disabled by default and should be explicitly enabled by `PPROF_ADDR`.

> [!IMPORTANT]  
> For configuring OpenTelemetry exporters, see [OpenTelemetry exporters][otel-exporter] documentation.

[otel-exporter]: https://opentelemetry.io/docs/specs/otel/protocol/exporter/

Metrics and pprof can be served from same address if needed, set both addresses to the same value.

### Example

#### Environment file
```bash
OTEL_LOG_LEVEL=debug
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
OTEL_EXPORTER_OTLP_INSECURE=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4317
OTEL_RESOURCE_ATTRIBUTES=service.name=go-faster.simon

# metrics exporter
OTEL_METRIC_EXPORT_INTERVAL=10000
OTEL_METRIC_EXPORT_TIMEOUT=5000

# pyroscope
PYROSCOPE_URL=http://127.0.0.1:4040
# should be same as service.name
PYROSCOPE_APP_NAME=go-faster.simon
PYROSCOPE_ENABLE=true

# use new metrics
OTEL_GO_X_DEPRECATED_RUNTIME_METRICS=false
# generate instance id
OTEL_GO_X_RESOURCE=true
```

#### Docker Compose
```yaml
services:
  app:
    image: ghcr.io/go-faster/simon:0.6.1
    environment:
      - OTEL_LOG_LEVEL=debug
      - OTEL_EXPORTER_OTLP_PROTOCOL=grpc
      - OTEL_EXPORTER_OTLP_INSECURE=true
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://otelcol:4317
      - OTEL_GO_X_DEPRECATED_RUNTIME_METRICS=false
      - OTEL_GO_X_RESOURCE=true
      - OTEL_METRIC_EXPORT_INTERVAL=1000
      - OTEL_METRIC_EXPORT_TIMEOUT=500
```

#### Kubernetes
```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: simon-client
  namespace: simon
spec:
  replicas: 1
  selector:
    matchLabels:
      app: simon-client
  template:
    metadata:
      labels:
        app: simon-client
    spec:
      containers:
        - name: ingest
          image: ghcr.io/go-faster/simon:0.6.1
          env:
            - name: OTEL_EXPORTER_OTLP_PROTOCOL
              value: "grpc"
            - name: OTEL_EXPORTER_OTLP_ENDPOINT
              value: "http://otel-collector.monitoring.svc.cluster.local:4317"
            - name: OTEL_LOG_LEVEL
              value: "debug"
            - name: OTEL_EXPORTER_OTLP_INSECURE
              value: "true"
            - name: OTEL_GO_X_DEPRECATED_RUNTIME_METRICS
              value: "false"
            - name: OTEL_METRIC_EXPORT_INTERVAL
              value: "1000"
            - name: OTEL_METRIC_EXPORT_TIMEOUT
              value: "500"
```

### Reference

| Name                                  | Description                      | Example                 | Default                |
|---------------------------------------|----------------------------------|-------------------------|------------------------|
| `AUTOMAXPROCS`                        | Use [automaxprocs][automaxprocs] | `0`                     | `1`                    |
| `AUTOMAXPROCS_MIN`                    | Minimum `GOMAXPROCS` to use      | `2`                     | `1`                    |
| `OTEL_RESOURCE_ATTRIBUTES`            | OTEL Resource attributes         | `service.name=app`      |                        |
| `OTEL_SERVICE_NAME`                   | OTEL Service name                | `app`                   | `unknown_service`      |
| `OTEL_EXPORTER_OTLP_PROTOCOL`         | OTLP protocol to use             | `http`                  | `grpc`                 |
| `OTEL_PROPAGATORS`                    | OTEL Propagators                 | `none`                  | `tracecontext,baggage` |
| `PPROF_ROUTES`                        | List of enabled pprof routes     | `cmdline,profile`       | See below              |
| `PPROF_ADDR`                          | Enable pprof and listen on addr  | `0.0.0.0:9010`          | N/A                    |
| `OTEL_LOG_LEVEL`                      | Log level                        | `debug`                 | `info`                 |
| `METRICS_ADDR`                        | Prometheus addr (fallback)       | `localhost:9464`        | Prometheus addr        |
| `OTEL_METRICS_EXPORTER`               | Metrics exporter to use          | `prometheus`            | `otlp`                 |
| `OTEL_EXPORTER_OTLP_METRICS_PROTOCOL` | Metrics OTLP protocol to use     | `http`                  | `grpc`                 |
| `OTEL_EXPORTER_PROMETHEUS_HOST`       | Host of prometheus addr          | `0.0.0.0`               | `localhost`            |
| `OTEL_EXPORTER_PROMETHEUS_PORT`       | Port of prometheus addr          | `9090`                  | `9464`                 |
| `OTEL_TRACES_EXPORTER`                | Traces exporter to use           | `otlp`                  | `otlp`                 |
| `OTEL_EXPORTER_OTLP_TRACES_PROTOCOL`  | Traces OTLP protocol to use      | `http`                  | `grpc`                 |
| `PYROSCOPE_ENABLE`                    | Enable Grafana Pyroscope         | `true`                  | `false`                |
| `PYROSCOPE_APP_NAME`                  | Pyroscope `ApplicationName`      | `app`                   |                        |
| `PYROSCOPE_URL`                       | Pyroscope `ServerAddress`        | `http://localhost:1234` |                        |
| `PYROSCOPE_USER`                      | Pyroscope `BasicAuthUser`        | `foo`                   |                        |
| `PYROSCOPE_PASSWORD`                  | Pyroscope `BasicAuthPassword`    | `bar`                   |                        |
| `PYROSCOPE_TENANT_ID`                 | Pyroscope `TenantID`             | `foo_bar`               |                        |

[automaxprocs]: https://github.com/uber-go/automaxprocs

### Metrics exporters

| Value        | Description                 |
|--------------|-----------------------------|
| `otlp`       | **OTLP exporter (default)** |
| `prometheus` | Prometheus exporter         |
| `none`       | No exporter                 |

### Trace exporters

| Value  | Description                 |
|--------|-----------------------------|
| `otlp` | **OTLP exporter (default)** |
| `none` | No exporter                 |



### Defaults

By default, OpenTelemetry SDK tries `localhost:4318` OTLP endpoint, assuming collector is running on the localhost.

If that is not true, following errors can be seen in the logs:

```json
{"error": "failed to upload metrics: Post \"https://localhost:4318/v1/metrics\": dial tcp 127.0.0.1:4318: connect: connection refused"}
```
```json
{"error": "failed to upload traces: Post \"https://localhost:4318/v1/traces\": dial tcp 127.0.0.1:4318: connect: connection refused"}
```

To fix that, configure exporters accordingly. For example, this will disable both metrics and traces exporters:

```bash
export OTEL_TRACES_EXPORTER="none"
export OTEL_METRICS_EXPORTER="none"
```

To enable Prometheus exporter, set `OTEL_METRICS_EXPORTER=prometheus` and `OTEL_EXPORTER_PROMETHEUS_HOST` and `OTEL_EXPORTER_PROMETHEUS_PORT` accordingly.

```bash
export OTEL_METRICS_EXPORTER="prometheus"
export OTEL_EXPORTER_PROMETHEUS_HOST="0.0.0.0"
export OTEL_EXPORTER_PROMETHEUS_PORT="9090"
```

### Routes for pprof

List of enabled pprof routes

**Name**: `PPROF_ROUTES`

**Default**: `profile,symbol,trace,goroutine,heap,threadcreate,block`

## Code coverage 

[![codecov](https://codecov.io/gh/go-faster/sdk/branch/main/graphs/sunburst.svg?token=cEE7AZ38Ho)](https://codecov.io/gh/go-faster/sdk)
