package zctx

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func BenchmarkWith(b *testing.B) {
	b.ReportAllocs()

	ctx := Base(context.Background(), zap.NewNop())

	f := zap.Int("foo", 1)

	for i := 0; i < b.N; i++ {
		c := With(ctx, f)
		_ = c.Done
	}
}

func BenchmarkFrom(b *testing.B) {
	ctx := Base(context.Background(), zap.NewNop())

	b.Run("Raw", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			lg := From(ctx)
			_ = lg.Sugar
		}
	})
	b.Run("TracedFresh", func(b *testing.B) {
		b.ReportAllocs()

		tracer := newTestTracer()
		ctx, span := tracer.Start(ctx, "test")
		defer span.End()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			lg := From(ctx)
			_ = lg.Sugar
		}
	})
	b.Run("TracedStarted", func(b *testing.B) {
		b.ReportAllocs()

		tracer := newTestTracer()
		ctx, span := tracer.Start(ctx, "test")
		defer span.End()

		ctx, lg := Start(ctx)
		useLogger(lg)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			useLogger(From(ctx))
		}
	})
	b.Run("TracedWith", func(b *testing.B) {
		b.ReportAllocs()

		tracer := newTestTracer()
		ctx, span := tracer.Start(ctx, "test")
		defer span.End()

		ctx = With(ctx, zap.Int("foo", 1))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			useLogger(From(ctx))
		}
	})
}

func useLogger(lg *zap.Logger) {
	_ = lg.Sugar
}

func BenchmarkTraceFields(b *testing.B) {
	ctx := context.Background()
	lg := zap.NewNop()

	b.Run("Prepared", func(b *testing.B) {
		b.ReportAllocs()
		tracer := newTestTracer()

		ctx, span := tracer.Start(ctx, "test")
		defer span.End()
		s := trace.SpanContextFromContext(ctx)
		traceIDField := zap.String("trace_id", s.TraceID().String())
		spanIDField := zap.String("span_id", s.SpanID().String())

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if v := trace.SpanContextFromContext(ctx); v.Equal(s) {
				nlg := lg.With(traceIDField, spanIDField)
				useLogger(nlg)
			} else {
				panic("?")
			}
		}
	})
	b.Run("Fresh", func(b *testing.B) {
		b.ReportAllocs()
		tracer := newTestTracer()

		ctx, span := tracer.Start(ctx, "test")
		defer span.End()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s := trace.SpanContextFromContext(ctx)
			nlg := lg.With(
				zap.String("trace_id", s.TraceID().String()),
				zap.String("span_id", s.SpanID().String()),
			)
			useLogger(nlg)
		}
	})
	b.Run("Equal", func(b *testing.B) {
		b.ReportAllocs()
		tracer := newTestTracer()

		ctx, span := tracer.Start(ctx, "test")
		defer span.End()
		s := trace.SpanContextFromContext(ctx)
		traceIDField := zap.String("trace_id", s.TraceID().String())
		spanIDField := zap.String("span_id", s.SpanID().String())

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if v := trace.SpanContextFromContext(ctx); v.Equal(s) {
				nlg := lg.With(traceIDField, spanIDField)
				useLogger(nlg)
			} else {
				panic("?")
			}
		}
	})
}
