// Package audit implements structured audit event recording and exporters.
package audit

import (
	"context"
	"sync"
	"time"

	"github.com/go-faster/errors"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/go-faster/sdk/zctx"
)

var (
	defaultClock       = func() time.Time { return time.Now() }
	defaultIDGenerator = uuid.NewString
)

// ShutdownFunc is a function that shuts down the Recorder.
type ShutdownFunc func(ctx context.Context) error

// Recorder records audit events.
type Recorder struct {
	exporters []Exporter
	redactor  Redactor
	lg        *zap.Logger
	conf      config
}

// New initializes a new Recorder.
func New(ctx context.Context, opts ...Option) (*Recorder, ShutdownFunc, error) {
	cfg := newConfig(opts)
	lg := zctx.From(ctx).Named("audit")
	if len(cfg.exporters) == 0 {
		cfg.exporters = []Exporter{NewNopExporter()}
	}
	r := &Recorder{exporters: cfg.exporters, redactor: cfg.redactor, lg: lg, conf: cfg}
	return r, r.Shutdown, nil
}

// Record records an event synchronously.
func (r *Recorder) Record(ctx context.Context, e Event) {
	if r == nil {
		return
	}
	if r.conf.traceEnrich {
		if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
			if e.TraceID == "" {
				e.TraceID = sc.TraceID().String()
			}
			if e.SpanID == "" {
				e.SpanID = sc.SpanID().String()
			}
		}
	}
	if r.redactor != nil {
		r.redactor.Redact(&e)
	}
	for _, exporter := range r.exporters {
		if exporter == nil {
			continue
		}
		if err := exporter.Export(ctx, []Event{e}); err != nil {
			r.lg.Error("audit export", zap.Error(err), zap.String("exporter", exporter.Name()))
		}
	}
}

// Emit builds and records an event.
func (r *Recorder) Emit(ctx context.Context, b *EventBuilder) {
	if r == nil || b == nil {
		return
	}
	if b.e.ID == "" && r.conf.idGenerator != nil {
		b.e.ID = r.conf.idGenerator()
	}
	if b.e.Time.IsZero() && r.conf.clock != nil {
		b.e.Time = r.conf.clock().UTC()
	}
	if b.e.Service == "" {
		b.e.Service = r.conf.service
	}
	if b.e.Component == "" {
		b.e.Component = r.conf.component
	}
	if b.e.Environment == "" {
		b.e.Environment = r.conf.environment
	}
	if b.e.Version == "" {
		b.e.Version = r.conf.version
	}
	e, err := b.Build(ctx)
	if err != nil {
		r.lg.Error("audit build", zap.Error(err))
		return
	}
	r.Record(ctx, e)
}

// Flush flushes pending events.
func (r *Recorder) Flush(ctx context.Context) error { return nil }

// Shutdown closes all exporters in parallel.
func (r *Recorder) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}
	var wg sync.WaitGroup
	errCh := make(chan error, len(r.exporters))
	wg.Add(len(r.exporters))
	for _, exporter := range r.exporters {
		exporter := exporter
		go func() {
			defer wg.Done()
			if exporter == nil {
				return
			}
			if err := exporter.Close(ctx); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// NewNop returns a no-op Recorder.
func NewNop() *Recorder {
	return &Recorder{exporters: []Exporter{NewNopExporter()}, redactor: NoRedact(), lg: zap.NewNop()}
}
