package otelsync_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/go-faster/sdk/otelsync"
)

func newProvider(t *testing.T) (*sdkmetric.MeterProvider, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })
	return mp, reader
}

func readMetrics(t *testing.T, mp *sdkmetric.MeterProvider, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, mp.ForceFlush(ctx))
	var data metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &data))
	return data
}

func TestGaugeInt64(t *testing.T) {
	mp, reader := newProvider(t)
	meter := mp.Meter("test")

	a := otelsync.NewAdapter(meter)
	g, err := a.GaugeInt64("test.gauge",
		metric.WithDescription("a test gauge"),
		metric.WithUnit("By"),
	)
	require.NoError(t, err)
	_, err = a.Register()
	require.NoError(t, err)

	g.Observe(42)

	data := readMetrics(t, mp, reader)
	require.Len(t, data.ScopeMetrics, 1)
	require.Len(t, data.ScopeMetrics[0].Metrics, 1)
	m := data.ScopeMetrics[0].Metrics[0]
	require.Equal(t, "test.gauge", m.Name)
	require.Equal(t, "a test gauge", m.Description)
	require.Equal(t, "By", m.Unit)

	gauge := m.Data.(metricdata.Gauge[int64])
	require.Len(t, gauge.DataPoints, 1)
	require.Equal(t, int64(42), gauge.DataPoints[0].Value)
}

func TestGaugeInt64_MultipleAttributes(t *testing.T) {
	mp, reader := newProvider(t)
	meter := mp.Meter("test")

	a := otelsync.NewAdapter(meter)
	g, err := a.GaugeInt64("test.gauge")
	require.NoError(t, err)
	_, err = a.Register()
	require.NoError(t, err)

	attrA := attribute.String("region", "us-east")
	attrB := attribute.String("region", "eu-west")
	g.Observe(10, metric.WithAttributes(attrA))
	g.Observe(20, metric.WithAttributes(attrB))
	// Overwrite first value.
	g.Observe(15, metric.WithAttributes(attrA))

	data := readMetrics(t, mp, reader)
	require.Len(t, data.ScopeMetrics[0].Metrics, 1)
	gauge := data.ScopeMetrics[0].Metrics[0].Data.(metricdata.Gauge[int64])
	require.Len(t, gauge.DataPoints, 2)

	vals := map[string]int64{}
	for _, dp := range gauge.DataPoints {
		v, _ := dp.Attributes.Value(attribute.Key("region"))
		vals[v.AsString()] = dp.Value
	}
	require.Equal(t, int64(15), vals["us-east"])
	require.Equal(t, int64(20), vals["eu-west"])
}

func TestGaugeFloat64(t *testing.T) {
	mp, reader := newProvider(t)
	meter := mp.Meter("test")

	a := otelsync.NewAdapter(meter)
	g, err := a.GaugeFloat64("test.gauge",
		metric.WithDescription("a float64 test gauge"),
		metric.WithUnit("1"),
	)
	require.NoError(t, err)
	_, err = a.Register()
	require.NoError(t, err)

	g.Observe(3.14)

	data := readMetrics(t, mp, reader)
	require.Len(t, data.ScopeMetrics, 1)
	require.Len(t, data.ScopeMetrics[0].Metrics, 1)
	m := data.ScopeMetrics[0].Metrics[0]
	require.Equal(t, "test.gauge", m.Name)
	require.Equal(t, "a float64 test gauge", m.Description)
	require.Equal(t, "1", m.Unit)

	gauge := m.Data.(metricdata.Gauge[float64])
	require.Len(t, gauge.DataPoints, 1)
	require.InDelta(t, 3.14, gauge.DataPoints[0].Value, 1e-9)
}

func TestGaugeFloat64_MultipleAttributes(t *testing.T) {
	mp, reader := newProvider(t)
	meter := mp.Meter("test")

	a := otelsync.NewAdapter(meter)
	g, err := a.GaugeFloat64("test.gauge")
	require.NoError(t, err)
	_, err = a.Register()
	require.NoError(t, err)

	attrA := attribute.String("env", "prod")
	attrB := attribute.String("env", "staging")
	g.Observe(0.9, metric.WithAttributes(attrA))
	g.Observe(0.5, metric.WithAttributes(attrB))
	g.Observe(0.95, metric.WithAttributes(attrA))

	data := readMetrics(t, mp, reader)
	gauge := data.ScopeMetrics[0].Metrics[0].Data.(metricdata.Gauge[float64])
	require.Len(t, gauge.DataPoints, 2)

	vals := map[string]float64{}
	for _, dp := range gauge.DataPoints {
		v, _ := dp.Attributes.Value(attribute.Key("env"))
		vals[v.AsString()] = dp.Value
	}
	require.InDelta(t, 0.95, vals["prod"], 1e-9)
	require.InDelta(t, 0.5, vals["staging"], 1e-9)
}

func TestAdapter_MixedGauges(t *testing.T) {
	mp, reader := newProvider(t)
	meter := mp.Meter("test")

	a := otelsync.NewAdapter(meter)
	gi, err := a.GaugeInt64("gauge.int")
	require.NoError(t, err)
	gf, err := a.GaugeFloat64("gauge.float")
	require.NoError(t, err)
	_, err = a.Register()
	require.NoError(t, err)

	gi.Observe(7)
	gf.Observe(1.5)

	data := readMetrics(t, mp, reader)
	require.Len(t, data.ScopeMetrics, 1)
	require.Len(t, data.ScopeMetrics[0].Metrics, 2)

	names := map[string]struct{}{}
	for _, m := range data.ScopeMetrics[0].Metrics {
		names[m.Name] = struct{}{}
	}
	require.Contains(t, names, "gauge.int")
	require.Contains(t, names, "gauge.float")
}
