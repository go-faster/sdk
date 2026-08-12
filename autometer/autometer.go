// Package autometer provides an OpenTelemetry MeterProvider creation
// function.
package autometer

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/go-faster/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"

	"github.com/go-faster/sdk/internal/autoenv"
	"github.com/go-faster/sdk/zctx"
)

const (
	expOTLP       = "otlp"
	expConsole    = "console"
	expNone       = autoenv.ExporterNone // no-op
	expPrometheus = "prometheus"

	protoHTTP         = "http"
	protoHTTPProtobuf = "http/protobuf"
	protoGRPC         = "grpc"
	defaultProto      = protoGRPC
)

const (
	writerStdout = "stdout"
	writerStderr = "stderr"
)

func writerByName(name string) io.Writer {
	switch name {
	case writerStdout, expConsole:
		return os.Stdout
	case writerStderr:
		return os.Stderr
	default:
		return io.Discard
	}
}

func getEnvOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func noopHandler(_ context.Context) error { return nil }

// ShutdownFunc is a function that shuts down the MeterProvider.
type ShutdownFunc func(ctx context.Context) error

// NewMeterProvider returns new metric.MeterProvider based on environment variables.
//
// OTEL_METRICS_EXPORTER is a comma-separated list of exporters, all of them are used.
func NewMeterProvider(ctx context.Context, options ...Option) (
	meterProvider metric.MeterProvider,
	meterShutdown ShutdownFunc,
	err error,
) {
	cfg := newConfig(options)
	lg := zctx.From(ctx)

	const envName = "OTEL_METRICS_EXPORTER"
	exporters, err := autoenv.ParseExporters(getEnvOr(envName, expOTLP))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "parse %s", envName)
	}
	if exporters[0] == expNone {
		lg.Debug("Using no-op metrics exporter")
		return noop.NewMeterProvider(), noopHandler, nil
	}

	var metricOptions []sdkmetric.Option
	if cfg.res != nil {
		metricOptions = append(metricOptions, sdkmetric.WithResource(cfg.res))
	}
	for _, exporter := range exporters {
		reader, err := newMetricReader(ctx, cfg, exporter)
		if err != nil {
			return nil, nil, err
		}
		metricOptions = append(metricOptions, sdkmetric.WithReader(reader))
	}

	provider := sdkmetric.NewMeterProvider(metricOptions...)
	return provider, provider.Shutdown, nil
}

func newMetricReader(ctx context.Context, cfg config, exporter string) (sdkmetric.Reader, error) {
	lg := zctx.From(ctx)
	switch exporter {
	case expPrometheus:
		lg.Debug("Using Prometheus metrics exporter")
		reg := cfg.prom
		if reg == nil {
			reg = prometheus.NewPedanticRegistry()
		}
		if cfg.promCallback != nil {
			switch v := reg.(type) {
			case *prometheus.Registry:
				cfg.promCallback(v)
			}
		}
		exp, err := otelprometheus.New(
			otelprometheus.WithRegisterer(reg),
		)
		if err != nil {
			return nil, errors.Wrap(err, "create Prometheus exporter")
		}
		// Register legacy prometheus-only runtime metrics for backward compatibility.
		reg.MustRegister(
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
			collectors.NewGoCollector(),
			collectors.NewBuildInfoCollector(),
		)
		return exp, nil
	case expOTLP:
		proto := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")
		if proto == "" {
			proto = os.Getenv("OTEL_EXPORTER_OTLP_METRICS_PROTOCOL")
		}
		if proto == "" {
			proto = defaultProto
		}
		lg.Debug("Using OTLP metrics exporter", zap.String("protocol", proto))
		switch proto {
		case protoHTTP, protoHTTPProtobuf:
			exp, err := otlpmetrichttp.New(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "create OTLP HTTP metric exporter")
			}
			return sdkmetric.NewPeriodicReader(exp), nil
		case protoGRPC:
			exp, err := otlpmetricgrpc.New(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "create OTLP gRPC metric exporter")
			}
			return sdkmetric.NewPeriodicReader(exp), nil
		default:
			return nil, errors.Errorf("unsupported metric OTLP protocol %q", proto)
		}
	case writerStdout, writerStderr, expConsole:
		lg.Debug("Using stdout metrics exporter", zap.String("writer", exporter))
		writer := cfg.writer
		if writer == nil {
			writer = writerByName(exporter)
		}
		enc := json.NewEncoder(writer)
		exp, err := stdoutmetric.New(stdoutmetric.WithEncoder(enc))
		if err != nil {
			return nil, errors.Wrapf(err, "create %q metric exporter", exporter)
		}
		return sdkmetric.NewPeriodicReader(exp), nil
	default:
		lookup := cfg.lookup
		if lookup == nil {
			break
		}
		lg.Debug("Looking for metrics exporter", zap.String("exporter", exporter))
		exp, ok, err := lookup(ctx, exporter)
		if err != nil {
			return nil, errors.Wrapf(err, "create %q", exporter)
		}
		if !ok {
			break
		}

		lg.Debug("Using user-defined metrics exporter", zap.String("exporter", exporter))
		return exp, nil
	}
	return nil, errors.Errorf("unsupported OTEL_METRICS_EXPORTER %q", exporter)
}
