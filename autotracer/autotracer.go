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
// Exporters that fail to initialize are logged and skipped, error is returned only
// if none of them could be set up.
//
// GOFASTER_OTLP_TRACES_ENDPOINTS (or GOFASTER_OTLP_ENDPOINTS) is a comma-separated
// list of additional OTLP endpoints to fan out spans to. Not defined by the
// OpenTelemetry specification, where OTEL_EXPORTER_OTLP_ENDPOINT is singular.
func NewTracerProvider(ctx context.Context, options ...Option) (
	tracerProvider trace.TracerProvider,
	tracerShutdown ShutdownFunc,
	err error,
) {
	cfg := newConfig(options)
	lg := zctx.From(ctx)

	const (
		signalName = "TRACES"
		envName    = "OTEL_" + signalName + "_EXPORTER"
	)
	exporters, err := autoenv.ParseExporters(getEnvOr(envName, expOTLP))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "parse %s", envName)
	}
	if exporters[0] == expNone {
		lg.Debug("Using no-op trace exporter")
		return noop.NewTracerProvider(), nop, nil
	}

	var (
		traceOptions []sdktrace.TracerProviderOption
		setupErrs    []error
		configured   int
	)
	if cfg.res != nil {
		traceOptions = append(traceOptions, sdktrace.WithResource(cfg.res))
	}
	addExporter := func(e sdktrace.SpanExporter) {
		traceOptions = append(traceOptions, sdktrace.WithBatcher(e))
		configured++
	}
	for _, exporter := range exporters {
		exp, err := newSpanExporter(ctx, cfg, exporter)
		if err != nil {
			// Single broken exporter should not take down the rest of them.
			lg.Warn("Failed to setup trace exporter",
				zap.String("exporter", exporter),
				zap.Error(err),
			)
			setupErrs = append(setupErrs, err)
			continue
		}
		addExporter(exp)
	}
	for _, endpoint := range autoenv.AdditionalEndpoints(signalName) {
		exp, err := newOTLPExporter(ctx, endpoint)
		if err != nil {
			lg.Warn("Failed to setup additional OTLP trace exporter",
				zap.String("endpoint", endpoint),
				zap.Error(err),
			)
			setupErrs = append(setupErrs, err)
			continue
		}
		lg.Debug("Using additional OTLP trace exporter", zap.String("endpoint", endpoint))
		addExporter(exp)
	}
	for _, exp := range cfg.additional {
		addExporter(exp)
	}
	if configured == 0 {
		return nil, nil, errors.Join(setupErrs...)
	}

	provider := sdktrace.NewTracerProvider(traceOptions...)
	return provider, provider.Shutdown, nil
}

// newOTLPExporter creates OTLP exporter, endpoint overrides OTEL_EXPORTER_OTLP_ENDPOINT if set.
func newOTLPExporter(ctx context.Context, endpoint string) (sdktrace.SpanExporter, error) {
	lg := zctx.From(ctx)

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
		var opts []otlptracehttp.Option
		switch {
		case endpoint == "":
		case strings.Contains(endpoint, "://"):
			opts = append(opts, otlptracehttp.WithEndpointURL(endpoint))
		default:
			opts = append(opts, otlptracehttp.WithEndpoint(endpoint))
		}
		exp, err := otlptracehttp.New(ctx, opts...)
		if err != nil {
			return nil, errors.Wrap(err, "create OTLP HTTP trace exporter")
		}
		return exp, nil
	case protoGRPC:
		var opts []otlptracegrpc.Option
		switch {
		case endpoint == "":
		case strings.Contains(endpoint, "://"):
			opts = append(opts, otlptracegrpc.WithEndpointURL(endpoint))
		default:
			opts = append(opts, otlptracegrpc.WithEndpoint(endpoint))
		}
		exp, err := otlptracegrpc.New(ctx, opts...)
		if err != nil {
			return nil, errors.Wrap(err, "create OTLP gRPC trace exporter")
		}
		return exp, nil
	default:
		return nil, errors.Errorf("unsupported traces otlp protocol %q", proto)
	}
}

func newSpanExporter(ctx context.Context, cfg config, exporter string) (sdktrace.SpanExporter, error) {
	lg := zctx.From(ctx)
	switch exporter {
	case expOTLP:
		return newOTLPExporter(ctx, "")
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
