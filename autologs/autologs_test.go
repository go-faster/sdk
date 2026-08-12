package autologs_test

import (
	"context"
	"io"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/go-faster/errors"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/go-faster/sdk/autologs"
	"github.com/go-faster/sdk/zctx"
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

func TestNewLoggerProviderMultipleExporters(t *testing.T) {
	ctx := context.Background()
	t.Setenv("OTEL_LOGS_EXPORTER", "first,second")
	ctx = zctx.Base(ctx, zaptest.NewLogger(t, zaptest.Level(zap.InfoLevel)))

	exporters := map[string]*testLogExporter{
		"first":  {},
		"second": {},
	}
	provider, shutdown, err := autologs.NewLoggerProvider(ctx,
		autologs.WithLookupExporter(func(ctx context.Context, name string) (sdklog.Exporter, bool, error) {
			exp, ok := exporters[name]
			if !ok {
				return nil, false, errors.Errorf("unexpected exporter %q", name)
			}
			return exp, true, nil
		}),
	)
	require.NoError(t, err)

	otelLg := zap.New(otelzap.NewCore("github.com/go-faster/sdk/app",
		otelzap.WithLoggerProvider(provider),
	))
	otelLg.Info("information")
	require.NoError(t, otelLg.Sync())
	require.NoError(t, shutdown(ctx))

	for name, exp := range exporters {
		records := exp.Records()
		require.Lenf(t, records, 1, "exporter %q", name)
		require.Equal(t, "information", records[0].Body().AsString())
	}
}

func TestNewLoggerProviderAdditionalExporters(t *testing.T) {
	ctx := zctx.Base(context.Background(), zaptest.NewLogger(t, zaptest.Level(zap.InfoLevel)))

	t.Run("Additional", func(t *testing.T) {
		t.Setenv("OTEL_LOGS_EXPORTER", "stdout")

		exporter := &testLogExporter{}
		provider, shutdown, err := autologs.NewLoggerProvider(ctx,
			autologs.WithWriter(io.Discard),
			autologs.WithAdditionalExporters(exporter),
		)
		require.NoError(t, err)

		otelLg := zap.New(otelzap.NewCore("test", otelzap.WithLoggerProvider(provider)))
		otelLg.Info("information")
		require.NoError(t, otelLg.Sync())
		require.NoError(t, shutdown(ctx))

		require.Len(t, exporter.Records(), 1)
	})
	t.Run("None", func(t *testing.T) {
		t.Setenv("OTEL_LOGS_EXPORTER", "none")

		exporter := &testLogExporter{}
		provider, shutdown, err := autologs.NewLoggerProvider(ctx,
			autologs.WithAdditionalExporters(exporter),
		)
		require.NoError(t, err)

		otelLg := zap.New(otelzap.NewCore("test", otelzap.WithLoggerProvider(provider)))
		otelLg.Info("information")
		require.NoError(t, otelLg.Sync())
		require.NoError(t, shutdown(ctx))

		require.Empty(t, exporter.Records())
	})
}

func TestNewLoggerProviderNegative(t *testing.T) {
	ctx := context.Background()
	for _, exp := range []string{
		"unsupported",
		"none,stdout",
		"stdout,none",
		",",
	} {
		t.Run(exp, func(t *testing.T) {
			t.Setenv("OTEL_LOGS_EXPORTER", exp)
			provider, shutdown, err := autologs.NewLoggerProvider(ctx)
			require.Error(t, err)
			require.Nil(t, provider)
			require.Nil(t, shutdown)
		})
	}
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
