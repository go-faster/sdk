// Package autotracer provides an OpenTelemetry TracerProvider creation
// function.
package autotracer

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/go-faster/errors"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	expOTLP   = "otlp"
	expNone   = "none" // no-op
	expJaeger = "jaeger"

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

func noop(_ context.Context) error { return nil }

type ShutdownFunc func(ctx context.Context) error

func NewTracerProvider(ctx context.Context, options ...Option) (
	tracerProvider trace.TracerProvider,
	tracerShutdown ShutdownFunc,
	err error,
) {
	cfg := newConfig(options)
	var traceOptions []sdktrace.TracerProviderOption
	if cfg.res != nil {
		traceOptions = append(traceOptions, sdktrace.WithResource(cfg.res))
	}
	ret := func(e sdktrace.SpanExporter) (trace.TracerProvider, func(ctx context.Context) error, error) {
		traceOptions = append(traceOptions, sdktrace.WithBatcher(e))
		return sdktrace.NewTracerProvider(traceOptions...), e.Shutdown, nil
	}
	switch exporter := strings.TrimSpace(getEnvOr("OTEL_TRACES_EXPORTER", expOTLP)); exporter {
	case expJaeger:
		exp, err := jaeger.New(jaeger.WithAgentEndpoint())
		if err != nil {
			return nil, nil, errors.Wrap(err, "jaeger")
		}
		return ret(exp)
	case expOTLP:
		proto := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")
		if proto == "" {
			proto = os.Getenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL")
		}
		if proto == "" {
			proto = defaultProto
		}
		switch proto {
		case protoGRPC:
			exp, err := otlptracegrpc.New(ctx)
			if err != nil {
				return nil, nil, errors.Errorf("failed to create trace exporter: %w", err)
			}
			return ret(exp)
		case protoHTTP:
			exp, err := otlptracehttp.New(ctx)
			if err != nil {
				return nil, nil, errors.Errorf("failed to create trace exporter: %w", err)
			}
			return ret(exp)
		default:
			return nil, nil, errors.Errorf("unsupported traces otlp protocol %q", proto)
		}
	case writerStdout, writerStderr:
		writer := cfg.writer
		if writer == nil {
			writer = writerByName(exporter)
		}
		exp, err := stdouttrace.New(stdouttrace.WithWriter(writer))
		if err != nil {
			return nil, nil, errors.Wrap(err, exporter)
		}
		return ret(exp)
	case expNone:
		return trace.NewNoopTracerProvider(), noop, nil
	default:
		return nil, nil, errors.Errorf("unsupported OTEL_TRACES_EXPORTER %q", exporter)
	}
}
