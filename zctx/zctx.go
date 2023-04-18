// Package zctx is a context-aware zap logger.
package zctx

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type key struct{}

var _nop = zap.NewNop()

type logger struct {
	base *zap.Logger

	// Not appending span, trace to base *zap.Logger so fields won't duplicate.
	//
	// See https://github.com/uber-go/zap/issues/765
	span  zap.Field
	trace zap.Field
}

func from(ctx context.Context) logger {
	v, ok := ctx.Value(key{}).(logger)
	if !ok {
		return logger{base: _nop}
	}
	return v
}

// From returns zap.Logger from context.
func From(ctx context.Context) *zap.Logger {
	lg := from(ctx)

	// Add or update trace and span IDs if available.
	if s := trace.SpanContextFromContext(ctx); s.IsValid() {
		const kTrace, kSpan = "trace_id", "span_id"

		// Hex-encoded lowercase as per OTEL log data model.
		traceID, spanID := s.TraceID().String(), s.SpanID().String()
		if lg.trace.Key == kTrace {
			lg.trace.String = traceID
		} else {
			lg.trace = zap.String(kTrace, traceID)
		}
		if lg.span.Key == kSpan {
			lg.span.String = spanID
		} else {
			lg.span = zap.String(kSpan, spanID)
		}
		return lg.base.With(lg.trace, lg.span)
	}

	return lg.base
}

// With returns new context with provided zap fields.
func With(ctx context.Context, fields ...zap.Field) context.Context {
	lg := from(ctx)
	lg.base = lg.base.With(fields...)
	return context.WithValue(ctx, key{}, lg)
}

// Base initializes root logger for using as base context.
func Base(ctx context.Context, lg *zap.Logger) context.Context {
	return context.WithValue(ctx, key{}, logger{base: lg})
}
