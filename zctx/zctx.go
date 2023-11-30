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
	// Base logger, should not contain span_id and trace_id fields.
	base *zap.Logger

	// Span-scoped logger that caches span_id and trace_id fields.
	//
	// Will be returned by From(ctx) if ctx contains the same span.
	lg   *zap.Logger
	span trace.SpanContext
}

func (l *logger) SetSpan(s trace.SpanContext) {
	l.span = s
	l.lg = l.base.With(
		zap.String("span_id", s.SpanID().String()),
		zap.String("trace_id", s.TraceID().String()),
	)
}

func from(ctx context.Context) logger {
	v, ok := ctx.Value(key{}).(logger)
	if !ok {
		return logger{base: _nop}
	}
	return v
}

// Start allocates new span logger and returns new context with it.
// Use Start to reduce allocations during From, caching the span-scoped logger.
//
// Should be same as ctx = With(ctx), but more effective.
func Start(ctx context.Context) (context.Context, *zap.Logger) {
	v := from(ctx)
	s := trace.SpanContextFromContext(ctx)
	if s.Equal(v.span) {
		return ctx, v.lg
	}
	if !s.IsValid() {
		return ctx, v.lg
	}

	v.SetSpan(s)
	return context.WithValue(ctx, key{}, v), v.lg
}

// From returns zap.Logger from context.
func From(ctx context.Context) *zap.Logger {
	v := from(ctx)
	s := trace.SpanContextFromContext(ctx)
	if v.lg != nil && s.Equal(v.span) {
		return v.lg
	}
	if !s.IsValid() {
		return v.base
	}
	v.SetSpan(s)
	return v.lg
}

func with(ctx context.Context, v logger) context.Context {
	return context.WithValue(ctx, key{}, v)
}

// With returns new context with provided zap fields.
//
// The span and trace IDs must not be added to the base logger because zap
// can't update or replace fields.
func With(ctx context.Context, fields ...zap.Field) context.Context {
	v := from(ctx)
	v.base = v.base.With(fields...)

	// Check that cached logger is from current span.
	s := trace.SpanContextFromContext(ctx)
	if v.lg != nil && s.Equal(v.span) {
		// Same span, updating cached logger with new fields.
		v.lg = v.lg.With(fields...)
	} else if s.IsValid() {
		// New span. Caching logger.
		//
		// Next call to From in same span
		// will return cached logger.
		v.SetSpan(s)
	} else {
		// Not in span anymore.
		v.lg = v.base
		v.span = s
	}

	return with(ctx, v)
}

// Base initializes root logger for using as a base context. Should be done early.
//
// The span and trace IDs must not be added to the base logger because zap
// can't update or replace fields.
func Base(ctx context.Context, lg *zap.Logger) context.Context {
	if lg == nil {
		lg = _nop
	}
	return with(ctx, logger{base: lg})
}
