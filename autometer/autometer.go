// Package autometer provides an OpenTelemetry MeterProvider creation
// function.
package autometer

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"

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

	"github.com/go-faster/sdk/zctx"
)

const (
	expOTLP       = "otlp"
	expNone       = "none" // no-op
	expPrometheus = "prometheus"

	protoHTTP    = "http"
	protoGRPC    = "grpc"
	defaultProto = protoGRPC
)

const (
	writerStdout = "stdout"
	writerStderr = "stderr"
)

func writerByName(name string) io.Writer {
	switch name {
	case writerStdout:
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
func NewMeterProvider(ctx context.Context, options ...Option) (
	meterProvider metric.MeterProvider,
	meterShutdown ShutdownFunc,
	err error,
) {
	cfg := newConfig(options)
	lg := zctx.From(ctx)
	var metricOptions []sdkmetric.Option
	if cfg.res != nil {
		metricOptions = append(metricOptions, sdkmetric.WithResource(cfg.res))
	}

	ret := func(r sdkmetric.Reader) (metric.MeterProvider, func(ctx context.Context) error, error) {
		metricOptions = append(metricOptions, sdkmetric.WithReader(r))
		return sdkmetric.NewMeterProvider(metricOptions...), r.Shutdown, nil
	}

	// Metrics exporter.
	exporter := strings.TrimSpace(getEnvOr("OTEL_METRICS_EXPORTER", expOTLP))
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
			return nil, nil, errors.Wrap(err, "create Prometheus exporter")
		}
		// Register legacy prometheus-only runtime metrics for backward compatibility.
		reg.MustRegister(
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
			collectors.NewGoCollector(),
			collectors.NewBuildInfoCollector(),
		)
		return ret(exp)
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
		case protoHTTP:
			exp, err := otlpmetrichttp.New(ctx)
			if err != nil {
				return nil, nil, errors.Wrap(err, "create OTLP HTTP metric exporter")
			}
			return ret(sdkmetric.NewPeriodicReader(exp))
		case protoGRPC:
			exp, err := otlpmetricgrpc.New(ctx)
			if err != nil {
				return nil, nil, errors.Wrap(err, "create OTLP gRPC metric exporter")
			}
			return ret(sdkmetric.NewPeriodicReader(exp))
		default:
			return nil, nil, errors.Errorf("unsupported metric OTLP protocol %q", proto)
		}
	case writerStdout, writerStderr:
		lg.Debug("Using stdout metrics exporter", zap.String("writer", exporter))
		writer := cfg.writer
		if writer == nil {
			writer = writerByName(exporter)
		}
		enc := json.NewEncoder(writer)
		exp, err := stdoutmetric.New(stdoutmetric.WithEncoder(enc))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "create %q metric exporter", exporter)
		}
		return ret(sdkmetric.NewPeriodicReader(exp))
	case expNone:
		lg.Debug("Using no-op metrics exporter")
		return noop.NewMeterProvider(), noopHandler, nil
	default:
		lookup := cfg.lookup
		if lookup == nil {
			break
		}
		lg.Debug("Looking for metrics exporter", zap.String("exporter", exporter))
		exp, ok, err := lookup(ctx, exporter)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "create %q", exporter)
		}
		if !ok {
			break
		}

		lg.Debug("Using user-defined metrics exporter", zap.String("exporter", exporter))
		return ret(exp)
	}
	return nil, nil, errors.Errorf("unsupported OTEL_METRICS_EXPORTER %q", exporter)
}
