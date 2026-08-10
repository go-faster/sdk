package audit

import (
	"context"
	"sync"
)

// MemoryExporter stores received [Event]s in memory. It is safe for concurrent
// use and is intended as a testing/mock [Exporter] for package consumers.
type MemoryExporter struct {
	mu     sync.Mutex
	events []Event
}

// NewMemoryExporter returns an empty [MemoryExporter].
func NewMemoryExporter() *MemoryExporter { return &MemoryExporter{} }

// Export implements [Exporter].
func (m *MemoryExporter) Export(_ context.Context, events []Event) error {
	m.mu.Lock()
	m.events = append(m.events, events...)
	m.mu.Unlock()
	return nil
}

// Close implements [Exporter].
func (m *MemoryExporter) Close(_ context.Context) error { return nil }

// Name implements [Exporter].
func (m *MemoryExporter) Name() string { return "memory" }

// Events returns a snapshot of the stored [Event]s.
func (m *MemoryExporter) Events() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Event, len(m.events))
	copy(out, m.events)
	return out
}

// Reset discards stored events.
func (m *MemoryExporter) Reset() {
	m.mu.Lock()
	m.events = nil
	m.mu.Unlock()
}

var _ Exporter = (*MemoryExporter)(nil)
