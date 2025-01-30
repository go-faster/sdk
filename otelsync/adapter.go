package otelsync

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

// Adapter provides a sync adapter over async metric instruments.
type Adapter struct {
	meter metric.Meter
	gauge []*GaugeInt64
}

func (a *Adapter) callback(_ context.Context, o metric.Observer) error {
	for _, v := range a.gauge {
		v.observe(o)
	}
	return nil
}

// Register registers callback.
func (a *Adapter) Register() (metric.Registration, error) {
	var in []metric.Observable
	for _, v := range a.gauge {
		in = append(in, v.Int64ObservableGauge)
	}
	return a.meter.RegisterCallback(a.callback, in...)
}

// GaugeInt64 returns a new sync int64 gauge. Register must be called after creating all gauges.
func (a *Adapter) GaugeInt64(name string, options ...metric.Int64ObservableGaugeOption) (metric.Int64Observer, error) {
	og, err := a.meter.Int64ObservableGauge(name, options...)
	if err != nil {
		return nil, err
	}
	g := &GaugeInt64{
		Int64ObservableGauge: og,
	}
	a.gauge = append(a.gauge, g)
	return g, nil
}

func NewAdapter(m metric.Meter) *Adapter {
	a := &Adapter{
		meter: m,
	}

	return a
}
