// Package autotracer provides an OpenTelemetry TracerProvider creation
// function.
package autotracer

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/go-faster/errors"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"github.com/go-faster/sdk/zctx"
)

const (
	expOTLP = "otlp"
	expNone = "none" // no-op

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

func nop(_ context.Context) error { return nil }

type ShutdownFunc func(ctx context.Context) error

func NewTracerProvider(ctx context.Context, options ...Option) (
	tracerProvider trace.TracerProvider,
	tracerShutdown ShutdownFunc,
	err error,
) {
	cfg := newConfig(options)
	lg := zctx.From(ctx)
	var traceOptions []sdktrace.TracerProviderOption
	if cfg.res != nil {
		traceOptions = append(traceOptions, sdktrace.WithResource(cfg.res))
	}
	ret := func(e sdktrace.SpanExporter) (trace.TracerProvider, func(ctx context.Context) error, error) {
		traceOptions = append(traceOptions, sdktrace.WithBatcher(e))
		provider := sdktrace.NewTracerProvider(traceOptions...)
		return provider, provider.Shutdown, nil
	}

	exporter := strings.TrimSpace(getEnvOr("OTEL_TRACES_EXPORTER", expOTLP))
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
				return nil, nil, errors.Wrap(err, "create OTLP HTTP trace exporter")
			}
			return ret(exp)
		case protoGRPC:
			exp, err := otlptracegrpc.New(ctx)
			if err != nil {
				return nil, nil, errors.Wrap(err, "create OTLP gRPC trace exporter")
			}
			return ret(exp)
		default:
			return nil, nil, errors.Errorf("unsupported traces otlp protocol %q", proto)
		}
	case writerStdout, writerStderr:
		lg.Debug("Using stdout trace exporter", zap.String("writer", exporter))
		writer := cfg.writer
		if writer == nil {
			writer = writerByName(exporter)
		}
		exp, err := stdouttrace.New(stdouttrace.WithWriter(writer))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "create %q trace exporter", exporter)
		}
		return ret(exp)
	case expNone:
		lg.Debug("Using no-op trace exporter")
		return noop.NewTracerProvider(), nop, nil
	default:
		lookup := cfg.lookup
		if lookup == nil {
			break
		}
		lg.Debug("Looking for traces exporter", zap.String("exporter", exporter))
		exp, ok, err := lookup(ctx, exporter)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "create %q", exporter)
		}
		if !ok {
			break
		}

		lg.Debug("Using user-defined traces exporter", zap.String("exporter", exporter))
		return ret(exp)
	}
	return nil, nil, errors.Errorf("unsupported OTEL_TRACES_EXPORTER %q", exporter)
}
