package autologs

import (
	"context"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/go-faster/sdk/zctx"
)

// Setup OpenTelemetry to zap logger bridge.
func Setup(ctx context.Context, loggerProvider log.LoggerProvider, teeCore bool) (context.Context, error) {
	lg := zctx.From(ctx)
	otelCore := otelzap.NewCore("github.com/go-faster/sdk/app",
		otelzap.WithLoggerProvider(loggerProvider),
	)
	wrapCore := func(core zapcore.Core) zapcore.Core {
		return otelCore // log only to bridge
	}
	if teeCore {
		wrapCore = func(core zapcore.Core) zapcore.Core {
			// Log both to bridge and original core.
			return zapcore.NewTee(core, otelCore)
		}
	}
	return zctx.Base(ctx,
		lg.WithOptions(
			zap.WrapCore(wrapCore),
		),
	), nil
}
