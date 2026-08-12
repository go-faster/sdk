package autometer_test

import (
	"context"
	"io"
	"testing"

	"github.com/go-faster/errors"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/go-faster/sdk/autometer"
)

func TestNewMeterProvider(t *testing.T) {
	ctx := context.Background()
	res := resource.Default()
	t.Run("Positive", func(t *testing.T) {
		t.Setenv("OTEL_METRICS_EXPORTER", "none")
		meter, stop, err := autometer.NewMeterProvider(ctx, autometer.WithResource(res))
		require.NoError(t, err)
		require.NotNil(t, meter)
		require.NotNil(t, stop)

		_ = meter.Meter("test")
		require.NoError(t, stop(ctx))
	})
	t.Run("Negative", func(t *testing.T) {
		for _, exp := range []string{
			"unsupported",
			"none,stdout",
			"stdout,none",
			",",
		} {
			t.Run(exp, func(t *testing.T) {
				t.Setenv("OTEL_METRICS_EXPORTER", exp)
				meter, stop, err := autometer.NewMeterProvider(ctx, autometer.WithResource(res))
				require.Error(t, err)
				require.Nil(t, meter)
				require.Nil(t, stop)
			})
		}
	})
	t.Run("Multiple", func(t *testing.T) {
		t.Setenv("OTEL_METRICS_EXPORTER", "first,second")

		readers := map[string]*sdkmetric.ManualReader{
			"first":  sdkmetric.NewManualReader(),
			"second": sdkmetric.NewManualReader(),
		}
		meter, stop, err := autometer.NewMeterProvider(ctx,
			autometer.WithResource(res),
			autometer.WithLookupExporter(func(ctx context.Context, name string) (sdkmetric.Reader, bool, error) {
				reader, ok := readers[name]
				if !ok {
					return nil, false, errors.Errorf("unexpected exporter %q", name)
				}
				return reader, true, nil
			}),
		)
		require.NoError(t, err)

		counter, err := meter.Meter("test").Int64Counter("test_counter")
		require.NoError(t, err)
		counter.Add(ctx, 1)

		for name, reader := range readers {
			var rm metricdata.ResourceMetrics
			require.NoErrorf(t, reader.Collect(ctx, &rm), "reader %q", name)
			require.Lenf(t, rm.ScopeMetrics, 1, "reader %q", name)
			require.Lenf(t, rm.ScopeMetrics[0].Metrics, 1, "reader %q", name)
			require.Equal(t, "test_counter", rm.ScopeMetrics[0].Metrics[0].Name)
		}
		require.NoError(t, stop(ctx))
	})
	t.Run("All", func(t *testing.T) {
		for _, exp := range []string{
			"none",
			"stdout",
			"stderr",
			"console",
			// "otlp", // TODO: add non-blocking dial
			"prometheus",
			"stdout,stderr",
			"stdout,stdout",
		} {
			t.Run(exp, func(t *testing.T) {
				t.Setenv("OTEL_METRICS_EXPORTER", exp)
				meter, stop, err := autometer.NewMeterProvider(ctx, autometer.WithResource(res), autometer.WithWriter(io.Discard))
				require.NoError(t, err)
				require.NotNil(t, meter)
				require.NotNil(t, stop)

				_ = meter.Meter("test")
				require.NoError(t, stop(ctx))
			})
		}
	})
}
