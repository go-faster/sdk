package autologs

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/go-faster/errors"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap"

	"github.com/go-faster/sdk/zctx"
)

const (
	expOTLP = "otlp"
	expNone = "none" // no-op

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

func nop(_ context.Context) error { return nil }

// ShutdownFunc is a function that shuts down the MeterProvider.
type ShutdownFunc func(ctx context.Context) error

// NewLoggerProvider initializes new [log.LoggerProvider] with the given options from environment variables.
func NewLoggerProvider(ctx context.Context, options ...Option) (
	meterProvider log.LoggerProvider,
	meterShutdown ShutdownFunc,
	err error,
) {
	cfg := newConfig(options)
	lg := zctx.From(ctx)
	var logOptions []sdklog.LoggerProviderOption
	if cfg.res != nil {
		logOptions = append(logOptions, sdklog.WithResource(cfg.res))
	}
	ret := func(e sdklog.Exporter) (log.LoggerProvider, func(ctx context.Context) error, error) {
		logOptions = append(logOptions, sdklog.WithProcessor(
			sdklog.NewBatchProcessor(e),
		))
		return sdklog.NewLoggerProvider(logOptions...), e.Shutdown, nil
	}
	exporter := strings.TrimSpace(getEnvOr("OTEL_LOGS_EXPORTER", expOTLP))
	switch exporter {
	case expOTLP:
		proto := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")
		if proto == "" {
			proto = os.Getenv("OTEL_EXPORTER_OTLP_LOGS_PROTOCOL")
		}
		if proto == "" {
			proto = defaultProto
		}
		lg.Debug("Using OTLP logs exporter", zap.String("protocol", proto))
		switch proto {
		case protoHTTP:
			exp, err := otlploghttp.New(ctx)
			if err != nil {
				return nil, nil, errors.Wrap(err, "create OTLP HTTP logs exporter")
			}
			return ret(exp)
		case protoGRPC:
			exp, err := otlploggrpc.New(ctx)
			if err != nil {
				return nil, nil, errors.Wrap(err, "create OTLP gRPC logs exporter")
			}
			return ret(exp)
		default:
			return nil, nil, errors.Errorf("unsupported logs otlp protocol %q", proto)
		}
	case writerStdout, writerStderr:
		lg.Debug("Using stdout log exporter", zap.String("writer", exporter))
		writer := cfg.writer
		if writer == nil {
			writer = writerByName(exporter)
		}
		exp, err := stdoutlog.New(stdoutlog.WithWriter(writer))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "create %q logs exporter", exporter)
		}
		return ret(exp)
	case expNone:
		lg.Debug("Using no-op logs exporter")
		return noop.NewLoggerProvider(), nop, nil
	default:
		lookup := cfg.lookup
		if lookup == nil {
			break
		}
		lg.Debug("Looking for logs exporter", zap.String("exporter", exporter))
		exp, ok, err := lookup(ctx, exporter)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "create %q", exporter)
		}
		if !ok {
			break
		}

		lg.Debug("Using user-defined log exporter", zap.String("exporter", exporter))
		return ret(exp)
	}
	return nil, nil, errors.Errorf("unsupported OTEL_LOGS_EXPORTER %q", exporter)
}
