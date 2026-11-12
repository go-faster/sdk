package audit

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/go-faster/errors"
)

// Exporter exports audit events.
type Exporter interface {
	Export(ctx context.Context, events []Event) error
	Close(ctx context.Context) error
	Name() string
}

// Chain creates a fan-out exporter.
func Chain(exporters ...Exporter) Exporter { return multiExporter(exporters) }

type multiExporter []Exporter

func (m multiExporter) Export(ctx context.Context, events []Event) error {
	for _, exporter := range m {
		if exporter == nil {
			continue
		}
		_ = exporter.Export(ctx, events)
	}
	return nil
}

func (m multiExporter) Close(ctx context.Context) error {
	var errs []error
	for _, exporter := range m {
		if exporter == nil {
			continue
		}
		if err := exporter.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (m multiExporter) Name() string { return "chain" }

// StdoutExporter exports JSON lines to a writer.
type StdoutExporter struct {
	out io.Writer
	mu  sync.Mutex
}

// NewStdoutExporter creates a StdoutExporter.
func NewStdoutExporter(w io.Writer) Exporter {
	if w == nil {
		w = io.Discard
	}
	return &StdoutExporter{out: w}
}

// Export implements Exporter.
func (s *StdoutExporter) Export(ctx context.Context, events []Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	enc := json.NewEncoder(s.out)
	for _, event := range events {
		if err := enc.Encode(event); err != nil {
			return errors.Wrap(err, "audit: encode json")
		}
	}
	return nil
}

// Close implements Exporter.
func (s *StdoutExporter) Close(ctx context.Context) error { return nil }

// Name implements Exporter.
func (s *StdoutExporter) Name() string { return "stdout" }

type nopExporter struct{}

// NewNopExporter creates a no-op exporter.
func NewNopExporter() Exporter { return nopExporter{} }

func (nopExporter) Export(ctx context.Context, events []Event) error { return nil }
func (nopExporter) Close(ctx context.Context) error                  { return nil }
func (nopExporter) Name() string                                     { return "nop" }
