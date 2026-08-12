package autologs

import (
	"context"
	"io"
	"os"

	"github.com/go-faster/errors"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

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

// ShutdownFunc is a function that shuts down the MeterProvider.
type ShutdownFunc func(ctx context.Context) error

// NewLoggerProvider initializes new [log.LoggerProvider] with the given options from environment variables.
//
// OTEL_LOGS_EXPORTER is a comma-separated list of exporters, all of them are used.
func NewLoggerProvider(ctx context.Context, options ...Option) (
	logProvider log.LoggerProvider,
	logShutdown ShutdownFunc,
	err error,
) {
	cfg := newConfig(options)
	lg := zctx.From(ctx)

	const envName = "OTEL_LOGS_EXPORTER"
	exporters, err := autoenv.ParseExporters(getEnvOr(envName, expOTLP))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "parse %s", envName)
	}
	if exporters[0] == expNone {
		lg.Debug("Using no-op logs exporter")
		return noop.NewLoggerProvider(), nop, nil
	}

	var logOptions []sdklog.LoggerProviderOption
	if cfg.res != nil {
		logOptions = append(logOptions, sdklog.WithResource(cfg.res))
	}
	severity := zapLevelToOTelSeverity(lg.Level())
	for _, exporter := range exporters {
		exp, err := newLogExporter(ctx, cfg, exporter)
		if err != nil {
			return nil, nil, err
		}
		logOptions = append(logOptions,
			sdklog.WithProcessor(&levelFilterProcessor{
				next:     sdklog.NewBatchProcessor(exp),
				severity: severity,
			}),
		)
	}

	provider := sdklog.NewLoggerProvider(logOptions...)
	return provider, provider.Shutdown, nil
}

func newLogExporter(ctx context.Context, cfg config, exporter string) (sdklog.Exporter, error) {
	lg := zctx.From(ctx)
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
		case protoHTTP, protoHTTPProtobuf:
			exp, err := otlploghttp.New(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "create OTLP HTTP logs exporter")
			}
			return exp, nil
		case protoGRPC:
			exp, err := otlploggrpc.New(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "create OTLP gRPC logs exporter")
			}
			return exp, nil
		default:
			return nil, errors.Errorf("unsupported logs otlp protocol %q", proto)
		}
	case writerStdout, writerStderr, expConsole:
		lg.Debug("Using stdout log exporter", zap.String("writer", exporter))
		writer := cfg.writer
		if writer == nil {
			writer = writerByName(exporter)
		}
		exp, err := stdoutlog.New(stdoutlog.WithWriter(writer))
		if err != nil {
			return nil, errors.Wrapf(err, "create %q logs exporter", exporter)
		}
		return exp, nil
	default:
		lookup := cfg.lookup
		if lookup == nil {
			break
		}
		lg.Debug("Looking for logs exporter", zap.String("exporter", exporter))
		exp, ok, err := lookup(ctx, exporter)
		if err != nil {
			return nil, errors.Wrapf(err, "create %q", exporter)
		}
		if !ok {
			break
		}

		lg.Debug("Using user-defined log exporter", zap.String("exporter", exporter))
		return exp, nil
	}
	return nil, errors.Errorf("unsupported OTEL_LOGS_EXPORTER %q", exporter)
}

// levelFilterProcessor implements level filtering, since otlplog does not.
//
// Fuck you too, OpenTelemetry.
type levelFilterProcessor struct {
	next     sdklog.Processor
	severity log.Severity
}

var _ sdklog.Processor = (*levelFilterProcessor)(nil)

// Enabled implements [sdklog.FilterProcessor].
func (l *levelFilterProcessor) Enabled(ctx context.Context, param sdklog.EnabledParameters) bool {
	return param.Severity >= l.severity
}

// OnEmit implements [sdklog.Processor].
func (l *levelFilterProcessor) OnEmit(ctx context.Context, record *sdklog.Record) error {
	return l.next.OnEmit(ctx, record)
}

// ForceFlush implements [sdklog.Processor].
func (l *levelFilterProcessor) ForceFlush(ctx context.Context) error {
	return l.next.ForceFlush(ctx)
}

// Shutdown implements [sdklog.Processor].
func (l *levelFilterProcessor) Shutdown(ctx context.Context) error {
	return l.next.Shutdown(ctx)
}

func zapLevelToOTelSeverity(level zapcore.Level) log.Severity {
	switch level {
	case zapcore.DebugLevel:
		return log.SeverityDebug
	case zapcore.InfoLevel:
		return log.SeverityInfo
	case zapcore.WarnLevel:
		return log.SeverityWarn
	case zapcore.ErrorLevel:
		return log.SeverityError
	case zapcore.DPanicLevel:
		return log.SeverityFatal1
	case zapcore.PanicLevel:
		return log.SeverityFatal2
	case zapcore.FatalLevel:
		return log.SeverityFatal3
	default:
		return log.SeverityUndefined
	}
}
