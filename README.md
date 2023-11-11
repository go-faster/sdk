# sdk [![Go Reference](https://img.shields.io/badge/go-pkg-00ADD8)](https://pkg.go.dev/github.com/go-faster/sdk#section-documentation) [![codecov](https://img.shields.io/codecov/c/github/go-faster/sdk?label=cover)](https://codecov.io/gh/go-faster/sdk) [![experimental](https://img.shields.io/badge/-experimental-blueviolet)](https://go-faster.org/docs/projects/status#experimental)

WIP SDK from go-faster for instrumentation.

## Packages

| Package      | Description                                             |
|--------------|---------------------------------------------------------|
| `autometer`  | Automatic OpenTelemetry MeterProvider from environment  |
| `autotracer` | Automatic OpenTelemetry TracerProvider from environment |
| `profiler`   | Explicit pprof routes                                   |
| `zctx`       | context.Context and tracing support for zap             |
| `gold`       | Golden files in tests                                   |
| `app`        | Automatic setup observability and run daemon            |

## Environment variables

> [!WARNING]
> The pprof listener is disabled by default and should be explicitly enabled by `PPROF_ADDR`.

> [!IMPORTANT]  
> For configuring OpenTelemetry exporters, see [OpenTelemetry exporters][otel-exporter] documentation.

[otel-exporter]: https://opentelemetry.io/docs/specs/otel/protocol/exporter/

Metrics and pprof can be served from same address if needed, set both addresses to the same value.

| Name                                  | Description                     | Example            | Default                |
|---------------------------------------|---------------------------------|--------------------|------------------------|
| `OTEL_RESOURCE_ATTRIBUTES`            | OTEL Resource attributes        | `service.name=app` |                        |
| `OTEL_SERVICE_NAME`                   | OTEL Service name               | `app`              | `unknown_service`      |
| `OTEL_EXPORTER_OTLP_PROTOCOL`         | OTLP protocol to use            | `http`             | `grpc`                 |
| `OTEL_PROPAGATORS`                    | OTEL Propagators                | `none`             | `tracecontext,baggage` |
| `PPROF_ROUTES`                        | List of enabled pprof routes    | `cmdline,profile`  | See below              |
| `PPROF_ADDR`                          | Enable pprof and listen on addr | `0.0.0.0:9010`     | N/A                    |
| `OTEL_LOG_LEVEL`                      | Log level                       | `debug`            | `info`                 |
| `METRICS_ADDR`                        | Prometheus addr (fallback)      | `localhost:9464`   | Prometheus addr        |
| `OTEL_METRICS_EXPORTER`               | Metrics exporter to use         | `prometheus`       | `otlp`                 |
| `OTEL_EXPORTER_OTLP_METRICS_PROTOCOL` | Metrics OTLP protocol to use    | `http`             | `grpc`                 |
| `OTEL_EXPORTER_PROMETHEUS_HOST`       | Host of prometheus addr         | `0.0.0.0`          | `localhost`            |
| `OTEL_EXPORTER_PROMETHEUS_PORT`       | Port of prometheus addr         | `9090`             | `9464`                 |
| `OTEL_TRACES_EXPORTER`                | Traces exporter to use          | `otlp`             | `otlp`                 |
| `OTEL_EXPORTER_OTLP_TRACES_PROTOCOL`  | Traces OTLP protocol to use     | `http`             | `grpc`                 |



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



## TODO
- [ ] Use slog
- [ ] Support for short-lived tasks
  - [ ] Metric, trace push
  - [ ] No need for http listeners for profiling
- [ ] Pyroscope compat


## Code coverage 

[![codecov](https://codecov.io/gh/go-faster/sdk/branch/main/graphs/sunburst.svg?token=cEE7AZ38Ho)](https://codecov.io/gh/go-faster/sdk)
