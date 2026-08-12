// Package autotracer provides an OpenTelemetry TracerProvider creation
// function.
package autotracer

import (
	"context"
	"io"
	"os"

	"github.com/go-faster/errors"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"github.com/go-faster/sdk/internal/autoenv"
	"github.com/go-faster/sdk/zctx"
)

const (
	expOTLP    = "otlp"
	expConsole = "console"
	expNone    = autoenv.ExporterNone // no-op

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

func nop(_ context.Context) error { return nil }

type ShutdownFunc func(ctx context.Context) error

// NewTracerProvider initializes new [trace.TracerProvider] with the given options
// from environment variables.
//
// OTEL_TRACES_EXPORTER is a comma-separated list of exporters, all of them are used.
func NewTracerProvider(ctx context.Context, options ...Option) (
	tracerProvider trace.TracerProvider,
	tracerShutdown ShutdownFunc,
	err error,
) {
	cfg := newConfig(options)
	lg := zctx.From(ctx)

	const envName = "OTEL_TRACES_EXPORTER"
	exporters, err := autoenv.ParseExporters(getEnvOr(envName, expOTLP))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "parse %s", envName)
	}
	if exporters[0] == expNone {
		lg.Debug("Using no-op trace exporter")
		return noop.NewTracerProvider(), nop, nil
	}

	var traceOptions []sdktrace.TracerProviderOption
	if cfg.res != nil {
		traceOptions = append(traceOptions, sdktrace.WithResource(cfg.res))
	}
	for _, exporter := range exporters {
		exp, err := newSpanExporter(ctx, cfg, exporter)
		if err != nil {
			return nil, nil, err
		}
		traceOptions = append(traceOptions, sdktrace.WithBatcher(exp))
	}
	for _, exp := range cfg.additional {
		traceOptions = append(traceOptions, sdktrace.WithBatcher(exp))
	}

	provider := sdktrace.NewTracerProvider(traceOptions...)
	return provider, provider.Shutdown, nil
}

func newSpanExporter(ctx context.Context, cfg config, exporter string) (sdktrace.SpanExporter, error) {
	lg := zctx.From(ctx)
	switch exporter {
	case expOTLP:
		proto := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")
		if proto == "" {
			proto = os.Getenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL")
		}
		if proto == "" {
			proto = defaultProto
		}
		lg.Debug("Using OTLP trace exporter", zap.String("protocol", proto))
		switch proto {
		case protoHTTP, protoHTTPProtobuf:
			exp, err := otlptracehttp.New(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "create OTLP HTTP trace exporter")
			}
			return exp, nil
		case protoGRPC:
			exp, err := otlptracegrpc.New(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "create OTLP gRPC trace exporter")
			}
			return exp, nil
		default:
			return nil, errors.Errorf("unsupported traces otlp protocol %q", proto)
		}
	case writerStdout, writerStderr, expConsole:
		lg.Debug("Using stdout trace exporter", zap.String("writer", exporter))
		writer := cfg.writer
		if writer == nil {
			writer = writerByName(exporter)
		}
		exp, err := stdouttrace.New(stdouttrace.WithWriter(writer))
		if err != nil {
			return nil, errors.Wrapf(err, "create %q trace exporter", exporter)
		}
		return exp, nil
	default:
		lookup := cfg.lookup
		if lookup == nil {
			break
		}
		lg.Debug("Looking for traces exporter", zap.String("exporter", exporter))
		exp, ok, err := lookup(ctx, exporter)
		if err != nil {
			return nil, errors.Wrapf(err, "create %q", exporter)
		}
		if !ok {
			break
		}

		lg.Debug("Using user-defined traces exporter", zap.String("exporter", exporter))
		return exp, nil
	}
	return nil, errors.Errorf("unsupported OTEL_TRACES_EXPORTER %q", exporter)
}
