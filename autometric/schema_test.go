package autometric

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"

	"github.com/go-faster/sdk/gold"
)

func TestMain(m *testing.M) {
	// Explicitly registering flags for golden files.
	gold.Init()

	os.Exit(m.Run())
}

type testStats struct {
	Float64Counter       metric.Float64Counter       `name:"float64counter" description:"Float64Counter test metric" unit:"Float64Counter units"`
	Float64Gauge         metric.Float64Gauge         `name:"float64gauge" description:"Float64Gauge test metric" unit:"Float64Gauge units"`
	Float64Histogram     metric.Float64Histogram     `name:"float64histogram" description:"Float64Histogram test metric" unit:"Float64Histogram units"`
	Float64UpDownCounter metric.Float64UpDownCounter `name:"float64updowncounter" description:"Float64UpDownCounter test metric" unit:"Float64UpDownCounter units"`
	Int64Counter         metric.Int64Counter         `name:"int64counter" description:"Int64Counter test metric" unit:"Int64Counter units"`
	Int64Gauge           metric.Int64Gauge           `name:"int64gauge" description:"Int64Gauge test metric" unit:"Int64Gauge units"`
	Int64Histogram       metric.Int64Histogram       `name:"int64histogram" description:"Int64Histogram test metric" unit:"Int64Histogram units"`
	Int64UpDownCounter   metric.Int64UpDownCounter   `name:"int64updowncounter" description:"Int64UpDownCounter test metric" unit:"Int64UpDownCounter units"`
}

func TestSchema(t *testing.T) {
	var stats testStats
	infos, err := Schema(&stats, InitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	gotMetric := map[string]MetricInfo{}
	for _, m := range infos.Metrics {
		gotMetric[m.Name] = m
	}

	expectedMetrics := map[string]MetricInfo{}
	for _, m := range []struct {
		name string
		typ  string
	}{
		{"Float64Counter", "sum"},
		{"Float64Gauge", "gauge"},
		{"Float64Histogram", "histogram"},
		{"Float64UpDownCounter", "sum"},
		{"Int64Counter", "sum"},
		{"Int64Gauge", "gauge"},
		{"Int64Histogram", "histogram"},
		{"Int64UpDownCounter", "sum"},
	} {
		name := strings.ToLower(m.name)
		instrument := strings.TrimPrefix(name, "float64")
		instrument = strings.TrimPrefix(instrument, "int64")

		mi := MetricInfo{
			Type:        m.typ,
			Instrument:  instrument,
			Name:        name,
			Description: m.name + " test metric",
			Unit:        m.name + " units",
		}
		expectedMetrics[mi.Name] = mi
	}

	require.Equal(t,
		expectedMetrics,
		gotMetric,
	)
}

func TestSchemaYAML(t *testing.T) {
	var stats testStats
	infos, err := Schema(&stats, InitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	gold.Str(t, string(infos.WeaverYAML()), "schema.yaml.golden")
}

type a1Stats struct {
	InflightRequests metric.Int64UpDownCounter `name:"http.inflight_requests" description:"Number of inflight requests" unit:"{requests}"`
	MemoryUsage      metric.Int64Gauge         `name:"http.memory_usage" description:"Memory usage" unit:"Bytes"`
}

type a2Stats struct {
	InsertedRecords metric.Int64Counter `name:"logs.inserted_records" description:"Number of inserted log records" unit:"{records}"`
	InsertedBytes   metric.Int64Counter `name:"inserted_bytes" description:"Total number of inserted bytes by signal" unit:"By"`
}

type dupStats struct {
	InsertedRecords metric.Int64Counter   `name:"logs.inserted_records" description:"Number of inserted log records" unit:"{records}"`
	RequestLatency  metric.Int64Histogram `name:"http.request_latency" description:"Latency of requests" unit:"ms"`
}

func TestRegistryYAML(t *testing.T) {
	oldRegisty := globalRegistry
	t.Cleanup(func() {
		globalRegistry = oldRegisty
	})
	globalRegistry = &Registry{}
	a1Schema := Define[a1Stats](InitOptions{Prefix: "client."})
	a2Schema := Define[a2Stats](InitOptions{})
	require.Panics(t, func() {
		Define[int](InitOptions{})
	})
	require.Panics(t, func() {
		Define[dupStats](InitOptions{})
	})

	test := metricnoop.NewMeterProvider().Meter("test")
	a1Schema(test)
	a2Schema(test)

	require.ElementsMatch(t,
		[]MetricInfo{
			{Type: "gauge", Instrument: "gauge", Name: "client.http.memory_usage", Description: "Memory usage", Unit: "Bytes"},
			{Type: "sum", Instrument: "updowncounter", Name: "client.http.inflight_requests", Description: "Number of inflight requests", Unit: "{requests}"},
			{Type: "sum", Instrument: "counter", Name: "inserted_bytes", Description: "Total number of inserted bytes by signal", Unit: "By"},
			{Type: "sum", Instrument: "counter", Name: "logs.inserted_records", Description: "Number of inserted log records", Unit: "{records}"},
		},
		Collect(),
	)

	gold.Str(t, string(WeaverYAML()), "global.yaml.golden")
}
