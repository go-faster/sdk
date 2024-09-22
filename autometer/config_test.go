package autometer

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestWithLookupExporter(t *testing.T) {
	var lookup LookupExporter = func(ctx context.Context, name string) (sdkmetric.Reader, bool, error) {
		switch name {
		case "return_something":
			r := sdkmetric.NewManualReader()
			return r, true, nil
		case "return_error":
			return nil, false, errors.New("test error")
		default:
			return nil, false, nil
		}
	}

	for i, tt := range []struct {
		name        string
		containsErr string
	}{
		{"return_something", ``},
		{"return_error", `test error`},
		{"return_not_exist", `unsupported OTEL_METRICS_EXPORTER "return_not_exist"`},
	} {
		tt := tt
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			t.Setenv("OTEL_METRICS_EXPORTER", tt.name)
			ctx := context.Background()

			_, _, err := NewMeterProvider(ctx, WithLookupExporter(lookup))
			if tt.containsErr != "" {
				require.ErrorContains(t, err, tt.containsErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
