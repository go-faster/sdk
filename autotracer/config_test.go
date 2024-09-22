package autotracer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestWithLookupExporter(t *testing.T) {
	var lookup LookupExporter = func(ctx context.Context, name string) (trace.SpanExporter, bool, error) {
		switch name {
		case "return_something":
			e, err := stdouttrace.New(stdouttrace.WithWriter(io.Discard))
			return e, true, err
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
		{"return_not_exist", `unsupported OTEL_TRACES_EXPORTER "return_not_exist"`},
	} {
		tt := tt
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			t.Setenv("OTEL_TRACES_EXPORTER", tt.name)
			ctx := context.Background()

			_, _, err := NewTracerProvider(ctx, WithLookupExporter(lookup))
			if tt.containsErr != "" {
				require.ErrorContains(t, err, tt.containsErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
