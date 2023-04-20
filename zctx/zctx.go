// Package zctx is a context-aware zap logger.
package zctx

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type key struct{}

var _nop = zap.NewNop()

func from(ctx context.Context) *zap.Logger {
	v, ok := ctx.Value(key{}).(*zap.Logger)
	if !ok {
		return _nop
	}
	return v
}

// From returns zap.Logger from context.
func From(ctx context.Context) *zap.Logger {
	lg := from(ctx)

	// Add or update trace and span IDs if available.
	if s := trace.SpanContextFromContext(ctx); s.IsValid() {
		return lg.With(
			// Hex-encoded lowercase as per OTEL log data model.
			zap.String("trace_id", s.TraceID().String()),
			zap.String("span_id", s.SpanID().String()),
		)
	}

	return lg
}

// With returns new context with provided zap fields.
//
// The span and trace IDs must not be added to the base logger because zap
// can't update or replace fields.
func With(ctx context.Context, fields ...zap.Field) context.Context {
	return context.WithValue(ctx, key{}, from(ctx).With(fields...))
}

// Base initializes root logger for using as base context. Should be done and early.
//
// The span and trace IDs must not be added to the base logger because zap
// can't update or replace fields.
func Base(ctx context.Context, lg *zap.Logger) context.Context {
	return context.WithValue(ctx, key{}, lg)
}
