package autometer_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
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
		t.Setenv("OTEL_METRICS_EXPORTER", "unsupported")
		meter, stop, err := autometer.NewMeterProvider(ctx, autometer.WithResource(res))
		require.Error(t, err)
		require.Nil(t, meter)
		require.Nil(t, stop)
	})
	t.Run("All", func(t *testing.T) {
		for _, exp := range []string{
			"none",
			"stdout",
			"stderr",
			// "otlp", // TODO: add non-blocking dial
			"prometheus",
		} {
			t.Run(exp, func(t *testing.T) {
				t.Setenv("OTEL_METRICS_EXPORTER", exp)
				meter, stop, err := autometer.NewMeterProvider(ctx, autometer.WithResource(res))
				require.NoError(t, err)
				require.NotNil(t, meter)
				require.NotNil(t, stop)

				_ = meter.Meter("test")
				require.NoError(t, stop(ctx))
			})
		}
	})
}
