package zapencoder

import (
	"context"
	"encoding/json"
	"io"

	"github.com/go-faster/jx"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

type ReflectedEncoder struct {
	json.Encoder
	io.Writer
}

func (e *ReflectedEncoder) Encode(v any) error {
	if ctx, ok := v.(context.Context); ok {
		span := trace.SpanContextFromContext(ctx)
		var enc jx.Encoder
		enc.Obj(func(e *jx.Encoder) {
			if span.IsValid() {
				enc.Field("span_id", func(e *jx.Encoder) {
					e.Str(span.SpanID().String())
				})
				enc.Field("trace_id", func(e *jx.Encoder) {
					e.Str(span.TraceID().String())
				})
			}
		})
		_, err := e.Writer.Write(enc.Bytes())
		return err
	}
	return e.Encoder.Encode(v)
}

func NewReflectedEncoder(w io.Writer) zapcore.ReflectedEncoder {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &ReflectedEncoder{Encoder: *enc, Writer: w}
}
