package audit_test

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"go.uber.org/zap/zaptest"

	"github.com/go-faster/sdk/audit"
	"github.com/go-faster/sdk/autologs"
	"github.com/go-faster/sdk/zctx"
)

func TestOTLPExporter(t *testing.T) {
	ctx := zctx.Base(context.Background(), zaptest.NewLogger(t))
	exp := &testLogExporter{}
	lp := sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewSimpleProcessor(exp)))
	auditExp, err := audit.NewOTLPExporter(ctx, audit.WithLoggerProvider(lp))
	require.NoError(t, err)
	require.NoError(t, auditExp.Export(ctx, []audit.Event{fullEvent()}))
	require.NoError(t, auditExp.Close(ctx))
	require.False(t, exp.shutdown.Load())
	require.NoError(t, lp.Shutdown(ctx))
	records := exp.Records()
	require.Len(t, records, 1)
	require.Equal(t, "User login", records[0].Body().AsString())
	require.Equal(t, "6", records[0].SeverityText())

	owned, err := audit.NewOTLPExporter(ctx, audit.WithOTLPOptions([]autologs.Option{}))
	require.NoError(t, err)
	require.NoError(t, owned.Close(ctx))
}

type testLogExporter struct {
	records    []sdklog.Record
	recordsMux sync.Mutex
	shutdown   atomic.Bool
}

var _ sdklog.Exporter = (*testLogExporter)(nil)

func (t *testLogExporter) Records() []sdklog.Record {
	t.recordsMux.Lock()
	r := slices.Clone(t.records)
	t.recordsMux.Unlock()
	return r
}

func (t *testLogExporter) Export(ctx context.Context, records []sdklog.Record) error {
	t.recordsMux.Lock()
	t.records = append(t.records, records...)
	t.recordsMux.Unlock()
	return nil
}

func (t *testLogExporter) ForceFlush(ctx context.Context) error { return nil }

func (t *testLogExporter) Shutdown(ctx context.Context) error {
	t.shutdown.Store(true)
	return nil
}
