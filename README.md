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

| Name                                  | Description                    | Example            | Default                |
|---------------------------------------|--------------------------------|--------------------|------------------------|
| `OTEL_RESOURCE_ATTRIBUTES`            | OTEL Resource attributes       | `service.name=app` |                        |
| `OTEL_SERVICE_NAME`                   | OTEL Service name              | `app`              | `unknown_service`      |
| `OTEL_EXPORTER_OTLP_PROTOCOL`         | OTLP protocol to use           | `http`             | `grpc`                 |
| `OTEL_PROPAGATORS`                    | OTEL Propagators               | `none`             | `tracecontext,baggage` |
| `PPROF_ROUTES`                        | List of enabled pprof routes   | `cmdline,profile`  | See below              |
| `OTEL_LOG_LEVEL`                      | Log level                      | `debug`            | `info`                 |
| `METRICS_ADDR`                        | Address with metrics and pprof | `localhost:9464`   | Prometheus addr        |
| `OTEL_METRICS_EXPORTER`               | Metrics exporter to use        | `prometheus`       | `none`                 |
| `OTEL_EXPORTER_OTLP_METRICS_PROTOCOL` | Metrics OTLP protocol to use   | `http`             | `grpc`                 |
| `OTEL_EXPORTER_PROMETHEUS_HOST`       | Host of prometheus addr        | `0.0.0.0`          | `localhost`            |
| `OTEL_EXPORTER_PROMETHEUS_PORT`       | Port of prometheus addr        | `9090`             | `9464`                 |
| `OTEL_TRACES_EXPORTER`                | Traces exporter to use         | `jaeger`           | `none`                 |
| `OTEL_EXPORTER_OTLP_TRACES_PROTOCOL`  | Traces OTLP protocol to use    | `http`             | `grpc`                 |
| `OTEL_EXPORTER_JAEGER_AGENT_HOST`     | Jaeger exporter host           | `jaeger.svc.local` | `localhost`            |
| `OTEL_EXPORTER_JAEGER_AGENT_PORT`     | Jaeger exporter port           | `6831`             | `6831`                 |

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
