package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap/zaptest"

	"github.com/go-faster/sdk/audit"
	"github.com/go-faster/sdk/zctx"
)

func TestTelemetryAudit(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "none")
	t.Setenv("OTEL_TRACES_EXPORTER", "none")
	t.Setenv("OTEL_METRICS_EXPORTER", "none")
	ctx := zctx.Base(context.Background(), zaptest.NewLogger(t))
	m, err := newTelemetry(ctx, ctx, zaptest.NewLogger(t), resource.Default(), nil, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, m.Audit())
	require.NotPanics(t, func() { m.Audit().Record(ctx, audit.Event{}) })
	m.shutdown(ctx)
}
