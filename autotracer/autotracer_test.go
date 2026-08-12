package autotracer_test

import (
	"context"
	"io"
	"testing"

	"github.com/go-faster/errors"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-faster/sdk/autotracer"
)

// forceFlush exports pending spans, since [tracetest.InMemoryExporter] resets records on shutdown.
func forceFlush(t *testing.T, ctx context.Context, provider trace.TracerProvider) {
	t.Helper()

	flusher, ok := provider.(interface {
		ForceFlush(ctx context.Context) error
	})
	require.True(t, ok)
	require.NoError(t, flusher.ForceFlush(ctx))
}

func TestNewTracerProvider(t *testing.T) {
	ctx := context.Background()
	t.Run("All", func(t *testing.T) {
		for _, exp := range []string{
			"none",
			"stdout",
			"stderr",
			"console",
			// "otlp", // TODO: add non-blocking dial
			"stdout,stderr",
			"stdout,stdout",
		} {
			t.Run(exp, func(t *testing.T) {
				t.Setenv("OTEL_TRACES_EXPORTER", exp)
				tracer, stop, err := autotracer.NewTracerProvider(ctx, autotracer.WithWriter(io.Discard))
				require.NoError(t, err)
				require.NotNil(t, tracer)
				require.NotNil(t, stop)

				_, span := tracer.Tracer("test").Start(ctx, "test")
				span.End()
				require.NoError(t, stop(ctx))
			})
		}
	})
	t.Run("Negative", func(t *testing.T) {
		for _, exp := range []string{
			"unsupported",
			"none,stdout",
			"stdout,none",
			",",
		} {
			t.Run(exp, func(t *testing.T) {
				t.Setenv("OTEL_TRACES_EXPORTER", exp)
				tracer, stop, err := autotracer.NewTracerProvider(ctx, autotracer.WithWriter(io.Discard))
				require.Error(t, err)
				require.Nil(t, tracer)
				require.Nil(t, stop)
			})
		}
	})
	t.Run("Additional", func(t *testing.T) {
		t.Setenv("OTEL_TRACES_EXPORTER", "stdout")

		exp := tracetest.NewInMemoryExporter()
		tracer, stop, err := autotracer.NewTracerProvider(ctx,
			autotracer.WithWriter(io.Discard),
			autotracer.WithAdditionalExporters(exp),
		)
		require.NoError(t, err)
		defer func() { require.NoError(t, stop(ctx)) }()

		_, span := tracer.Tracer("test").Start(ctx, "test")
		span.End()
		forceFlush(t, ctx, tracer)

		require.Len(t, exp.GetSpans(), 1)
	})
	t.Run("AdditionalNone", func(t *testing.T) {
		t.Setenv("OTEL_TRACES_EXPORTER", "none")

		exp := tracetest.NewInMemoryExporter()
		tracer, stop, err := autotracer.NewTracerProvider(ctx,
			autotracer.WithAdditionalExporters(exp),
		)
		require.NoError(t, err)

		_, span := tracer.Tracer("test").Start(ctx, "test")
		span.End()
		require.NoError(t, stop(ctx))

		require.Empty(t, exp.GetSpans())
	})
	t.Run("Multiple", func(t *testing.T) {
		t.Setenv("OTEL_TRACES_EXPORTER", "first,second")

		exporters := map[string]*tracetest.InMemoryExporter{
			"first":  tracetest.NewInMemoryExporter(),
			"second": tracetest.NewInMemoryExporter(),
		}
		tracer, stop, err := autotracer.NewTracerProvider(ctx,
			autotracer.WithLookupExporter(func(ctx context.Context, name string) (sdktrace.SpanExporter, bool, error) {
				exp, ok := exporters[name]
				if !ok {
					return nil, false, errors.Errorf("unexpected exporter %q", name)
				}
				return exp, true, nil
			}),
		)
		require.NoError(t, err)
		defer func() { require.NoError(t, stop(ctx)) }()

		_, span := tracer.Tracer("test").Start(ctx, "test")
		span.End()
		forceFlush(t, ctx, tracer)

		for name, exp := range exporters {
			spans := exp.GetSpans()
			require.Lenf(t, spans, 1, "exporter %q", name)
			require.Equal(t, "test", spans[0].Name)
		}
	})
}
