package otelattr

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// WrapTracerProvider returns a provider whose spans carry edit's result
// wherever the instrumentation sets attributes: the options passed to
// [go.opentelemetry.io/otel/trace.Tracer.Start], [go.opentelemetry.io/otel/trace.Span.SetAttributes],
// span events, and links.
//
// A nil editor returns tp unchanged.
func WrapTracerProvider(tp trace.TracerProvider, edit Editor) trace.TracerProvider {
	if edit == nil {
		return tp
	}
	return &tracerProvider{TracerProvider: tp, edit: edit}
}

type tracerProvider struct {
	trace.TracerProvider

	edit Editor
}

func (p *tracerProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return &tracer{Tracer: p.TracerProvider.Tracer(name, opts...), provider: p}
}

type tracer struct {
	trace.Tracer

	provider *tracerProvider
}

func (t *tracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	cfg := trace.NewSpanStartConfig(opts...)
	edited := []trace.SpanStartOption{
		trace.WithAttributes(t.provider.edit.apply(cfg.Attributes())...),
		trace.WithSpanKind(cfg.SpanKind()),
		trace.WithLinks(t.provider.editLinks(cfg.Links())...),
	}
	if ts := cfg.Timestamp(); !ts.IsZero() {
		edited = append(edited, trace.WithTimestamp(ts))
	}
	if cfg.NewRoot() {
		edited = append(edited, trace.WithNewRoot())
	}

	ctx, s := t.Tracer.Start(ctx, name, edited...)
	// The context has to carry the wrapper too, or instrumentation reaching
	// for the current span via trace.SpanFromContext sets attributes on the
	// span underneath and skips the editor.
	wrapped := &span{Span: s, provider: t.provider}
	return trace.ContextWithSpan(ctx, wrapped), wrapped
}

type span struct {
	trace.Span

	provider *tracerProvider
}

func (s *span) SetAttributes(kv ...attribute.KeyValue) {
	s.Span.SetAttributes(s.provider.edit.apply(kv)...)
}

func (s *span) AddEvent(name string, opts ...trace.EventOption) {
	s.Span.AddEvent(name, s.provider.editEvent(trace.NewEventConfig(opts...))...)
}

// RecordError is an event, and an instrumentation that puts a payload on one
// puts it here too.
func (s *span) RecordError(err error, opts ...trace.EventOption) {
	if err == nil {
		return
	}
	s.Span.RecordError(err, s.provider.editEvent(trace.NewEventConfig(opts...))...)
}

func (s *span) AddLink(link trace.Link) {
	s.Span.AddLink(s.provider.editLink(link))
}

// TracerProvider returns the wrapping provider, so a span started from this
// span's provider is edited as well.
func (s *span) TracerProvider() trace.TracerProvider { return s.provider }

func (p *tracerProvider) editEvent(cfg trace.EventConfig) []trace.EventOption {
	opts := []trace.EventOption{
		trace.WithAttributes(p.edit.apply(cfg.Attributes())...),
		trace.WithStackTrace(cfg.StackTrace()),
	}
	if ts := cfg.Timestamp(); !ts.IsZero() {
		opts = append(opts, trace.WithTimestamp(ts))
	}
	return opts
}

func (p *tracerProvider) editLink(link trace.Link) trace.Link {
	link.Attributes = p.edit.apply(link.Attributes)
	return link
}

func (p *tracerProvider) editLinks(links []trace.Link) []trace.Link {
	if len(links) == 0 {
		return nil
	}
	out := make([]trace.Link, len(links))
	for i, l := range links {
		out[i] = p.editLink(l)
	}
	return out
}
