package otelsync

import (
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/embedded"
)

// GaugeInt64 is a wrapper around metric.Int64ObservableGauge that stores last value
// for each attribute set, providing a sync adapter over async gauge.
type GaugeInt64 struct {
	metric.Int64ObservableGauge
	embedded.Int64Observer

	mux    sync.Mutex
	values map[attribute.Set]int64
}

// Observe records a last value for attribute set.
func (g *GaugeInt64) Observe(v int64, options ...metric.ObserveOption) {
	g.mux.Lock()
	defer g.mux.Unlock()

	if g.values == nil {
		g.values = make(map[attribute.Set]int64)
	}

	g.values[metric.NewObserveConfig(options).Attributes()] = v
}

func (g *GaugeInt64) observe(o metric.Observer) {
	g.mux.Lock()
	defer g.mux.Unlock()

	for k, v := range g.values {
		o.ObserveInt64(g.Int64ObservableGauge, v, metric.WithAttributes(k.ToSlice()...))
	}
}
