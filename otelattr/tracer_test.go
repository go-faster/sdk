package otelattr_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-faster/sdk/otelattr"
)

func recorded(t *testing.T, edit otelattr.Editor, fn func(tr trace.Tracer)) sdktrace.ReadOnlySpan {
	t.Helper()

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { require.NoError(t, tp.Shutdown(context.Background())) })

	fn(otelattr.WrapTracerProvider(tp, edit).Tracer("test"))

	ended := sr.Ended()
	require.Len(t, ended, 1)
	return ended[0]
}

func TestWrapTracerProviderEditsStartAttributes(t *testing.T) {
	span := recorded(t, otelattr.RedactURL(), func(tr trace.Tracer) {
		_, s := tr.Start(context.Background(), "HTTP GET", trace.WithAttributes(
			attribute.String("url.full", "https://example.com/blob/key?sig=deadbeef"),
			attribute.String("server.address", "example.com"),
		))
		s.End()
	})

	require.Equal(t, []attribute.KeyValue{
		attribute.String("url.full", "https://example.com"),
		attribute.String("server.address", "example.com"),
	}, span.Attributes())
}

func TestWrapTracerProviderEditsSetAttributes(t *testing.T) {
	span := recorded(t, otelattr.Drop("secret"), func(tr trace.Tracer) {
		_, s := tr.Start(context.Background(), "op")
		s.SetAttributes(attribute.String("secret", "hunter2"), attribute.Int("size", 3))
		s.End()
	})

	require.Equal(t, []attribute.KeyValue{attribute.Int("size", 3)}, span.Attributes())
}

func TestWrapTracerProviderPreservesStartConfig(t *testing.T) {
	ts := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	linked := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1}, SpanID: trace.SpanID{2},
	})

	span := recorded(t, otelattr.Drop("secret"), func(tr trace.Tracer) {
		_, s := tr.Start(context.Background(), "op",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithTimestamp(ts),
			trace.WithNewRoot(),
			trace.WithLinks(trace.Link{
				SpanContext: linked,
				Attributes: []attribute.KeyValue{
					attribute.String("secret", "hunter2"),
					attribute.String("kept", "yes"),
				},
			}),
		)
		s.End()
	})

	require.Equal(t, trace.SpanKindClient, span.SpanKind())
	require.Equal(t, ts, span.StartTime())
	require.False(t, span.Parent().IsValid())
	require.Len(t, span.Links(), 1)
	require.Equal(t, linked, span.Links()[0].SpanContext)
	require.Equal(t, []attribute.KeyValue{attribute.String("kept", "yes")}, span.Links()[0].Attributes)
}

func TestWrapTracerProviderEditsEvents(t *testing.T) {
	span := recorded(t, otelattr.Drop("secret"), func(tr trace.Tracer) {
		_, s := tr.Start(context.Background(), "op")
		s.AddEvent("read", trace.WithAttributes(
			attribute.String("secret", "hunter2"),
			attribute.Int("bytes", 7),
		))
		s.RecordError(errors.New("boom"), trace.WithAttributes(attribute.String("secret", "hunter2")))
		s.AddLink(trace.Link{
			SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
				TraceID: trace.TraceID{3}, SpanID: trace.SpanID{4},
			}),
			Attributes: []attribute.KeyValue{attribute.String("secret", "hunter2")},
		})
		s.End()
	})

	events := span.Events()
	require.Len(t, events, 2)
	require.Equal(t, []attribute.KeyValue{attribute.Int("bytes", 7)}, events[0].Attributes)
	for _, kv := range events[1].Attributes {
		require.NotEqual(t, attribute.Key("secret"), kv.Key)
	}
	require.Len(t, span.Links(), 1)
	require.Empty(t, span.Links()[0].Attributes)
}

// A span hands its own provider to instrumentation that starts a child from it,
// so the wrapper has to survive that hop.
func TestWrapTracerProviderChildSpansAreEdited(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { require.NoError(t, tp.Shutdown(context.Background())) })

	ctx, parent := otelattr.WrapTracerProvider(tp, otelattr.Drop("secret")).Tracer("test").
		Start(context.Background(), "parent")
	_, child := trace.SpanFromContext(ctx).TracerProvider().Tracer("child").Start(ctx, "child",
		trace.WithAttributes(attribute.String("secret", "hunter2")))
	child.End()
	parent.End()

	ended := sr.Ended()
	require.Len(t, ended, 2)
	require.Equal(t, "child", ended[0].Name())
	require.Empty(t, ended[0].Attributes())
}

func TestWrapTracerProviderNilEditor(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	require.Equal(t, trace.TracerProvider(tp), otelattr.WrapTracerProvider(tp, nil))
}
