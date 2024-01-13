package autologs

import (
	"context"
	"os"
	"strings"

	"github.com/go-faster/errors"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/go-faster/sdk/zapotel"
	"github.com/go-faster/sdk/zctx"
)

// Setup OTLP log exporter if configured.
func Setup(ctx context.Context, res *resource.Resource) (context.Context, error) {
	if os.Getenv("OTEL_LOGS_EXPORTER") != "otlp" {
		return ctx, nil
	}
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}
	if endpoint == "" {
		endpoint = "localhost:4317"
	}
	endpoint = strings.TrimPrefix(endpoint, "http://")
	conn, err := grpc.DialContext(ctx, endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return ctx, errors.Wrap(err, "dial logs endpoint")
	}
	lg := zctx.From(ctx)
	otelCore := zapotel.New(lg.Level(), res, plogotlp.NewGRPCClient(conn))
	// Update logger down the stack.
	lg.Info("Setting up OTLP log exporter")
	lg = lg.WithOptions(
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewTee(core, otelCore)
		}),
	)
	return zctx.Base(ctx, lg), nil
}
