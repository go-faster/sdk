package autologs_test

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/go-faster/errors"
	"github.com/go-faster/sdk/autologs"
	"github.com/go-faster/sdk/zctx"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestNewLoggerProviderLevel(t *testing.T) {
	ctx := context.Background()
	const testExporterName = "amongus"
	t.Setenv("OTEL_LOGS_EXPORTER", testExporterName)

	baseLogger := zaptest.NewLogger(t, zaptest.Level(zap.InfoLevel))
	ctx = zctx.Base(ctx, baseLogger)

	exporter := &testLogExporter{}
	provider, shutdown, err := autologs.NewLoggerProvider(ctx, autologs.WithLookupExporter(func(ctx context.Context, name string) (sdklog.Exporter, bool, error) {
		if name != testExporterName {
			return nil, false, errors.Errorf("wrong exporter %q", name)
		}
		return exporter, true, nil
	}))
	require.NoError(t, err)

	otelCore := otelzap.NewCore("github.com/go-faster/sdk/app",
		otelzap.WithLoggerProvider(provider),
	)
	otelLg := zap.New(otelCore)
	otelLg.Debug("hot and lonely GPUs around you")
	otelLg.Info("information")
	otelLg.Warn("warning")

	require.NoError(t, otelLg.Sync())
	require.NoError(t, shutdown(ctx))
	require.True(t, exporter.shutdown.Load())

	var msgs []string
	for _, r := range exporter.Records() {
		msgs = append(msgs, r.Body().AsString())
	}
	require.Equal(t,
		[]string{
			"information",
			"warning",
		},
		msgs,
	)
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

// Export implements [sdklog.Exporter].
func (t *testLogExporter) Export(ctx context.Context, records []sdklog.Record) error {
	t.recordsMux.Lock()
	t.records = append(t.records, records...)
	t.recordsMux.Unlock()
	return nil
}

// ForceFlush implements [sdklog.Exporter].
func (t *testLogExporter) ForceFlush(ctx context.Context) error {
	return nil
}

// Shutdown implements [sdklog.Exporter].
func (t *testLogExporter) Shutdown(ctx context.Context) error {
	t.shutdown.Store(true)
	return nil
}
