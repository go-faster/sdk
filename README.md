# sdk [![Go Reference](https://img.shields.io/badge/go-pkg-00ADD8)](https://pkg.go.dev/github.com/go-faster/sdk#section-documentation) [![codecov](https://img.shields.io/codecov/c/github/go-faster/sdk?label=cover)](https://codecov.io/gh/go-faster/sdk) [![alpha](https://img.shields.io/badge/-experimental-blueviole)](https://go-faster.org/docs/projects/status#experimental)

WIP SDK from go-faster for instrumentation.

## Packages

| Package      | Description                                             |
|--------------|---------------------------------------------------------|
| `autometer`  | Automatic OpenTelemetry MeterProvider from environment  |
| `autotracer` | Automatic OpenTelemetry TracerProvider from environment |
| `profiler`   | Explicit pprof routes                                   |
| `zctx`       | context.Context and tracing support for zap             |

## TODO
- [ ] Use slog


## Code coverage 

[![codecov](https://codecov.io/gh/go-faster/sdk/branch/main/graphs/sunburst.svg?token=cEE7AZ38Ho)](https://codecov.io/gh/go-faster/sdk)
