package zapencoder

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

func hasField(fields []zapcore.Field, key string) bool {
	for _, f := range fields {
		if f.Key == key {
			return true
		}
	}
	return false
}

const (
	traceIDField = "trace_id"
	spanIDField  = "span_id"
)

func addFields(enc zapcore.ObjectEncoder, fields []zapcore.Field) {
	for _, f := range fields {
		if f.Interface != nil && f.Type == zapcore.StringerType {
			// Test for ctx.
			ctx, ok := f.Interface.(context.Context)
			if ok {
				spanCtx := trace.SpanContextFromContext(ctx)
				if spanCtx.IsValid() {
					if !hasField(fields, traceIDField) {
						enc.AddString(traceIDField, spanCtx.TraceID().String())
					}
					if !hasField(fields, spanIDField) {
						enc.AddString(spanIDField, spanCtx.SpanID().String())
					}
				}

				continue
			}
		}
		f.AddTo(enc)
	}
}
