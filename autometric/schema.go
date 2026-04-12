package autometric

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/go-faster/errors"
	"github.com/go-faster/yaml"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"

	"github.com/go-faster/sdk/internal/weaveryaml"
)

// Registry contains registered schemas for structs.
type Registry struct {
	ids map[string]MetricInfo
	mux sync.Mutex
}

// Register registers schema for struct s with given options.
func (r *Registry) Register(s any, opts InitOptions) error {
	sch, err := Schema(s, opts)
	if err != nil {
		return errors.Wrap(err, "register schema")
	}
	return r.RegisterMetrics(sch.Metrics)
}

// RegisterMetrics registers given metrics in registry.
func (r *Registry) RegisterMetrics(metrics []MetricInfo) error {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.ids == nil {
		r.ids = make(map[string]MetricInfo)
	}

	for i, m := range metrics {
		id := m.Name
		if _, exists := r.ids[id]; exists {
			// If metric with the same id already exists, we need to remove all ids from current schema
			// to avoid leaving registry in inconsistent state.
			for _, m := range metrics[:i] {
				delete(r.ids, m.Name)
			}
			return errors.Errorf("metric with id %q already exists", id)
		}
		r.ids[id] = m
	}
	return nil
}

// Collect returns all registered schemas.
func (r *Registry) Collect() []MetricInfo {
	r.mux.Lock()
	defer r.mux.Unlock()

	mi := make([]MetricInfo, 0, len(r.ids))
	for _, m := range r.ids {
		mi = append(mi, m)
	}
	return mi
}

// WeaverYAML converts all registered schemas in global registry to Weaver YAML schema format.
func (r *Registry) WeaverYAML() []byte {
	return weaverYAMLFromSchemas(r.Collect())
}

var globalRegistry = &Registry{}

// Register registers schema for struct s with given options in global registry.
func Register(s any, opts InitOptions) {
	if err := globalRegistry.Register(s, opts); err != nil {
		panic(err)
	}
}

// Collect returns all registered schemas from global registry.
func Collect() []MetricInfo {
	return globalRegistry.Collect()
}

// WeaverYAML converts all registered schemas in global registry to Weaver YAML schema format.
func WeaverYAML() []byte {
	return globalRegistry.WeaverYAML()
}

// Define registers schema for struct of type T with given options and returns function that
// initializes metrics of type T with given meter and options.
func Define[T any](opts InitOptions) func(meter metric.Meter) (T, error) {
	{
		t := new(T)
		Register(t, opts)
	}
	return func(meter metric.Meter) (t T, err error) {
		err = Init(meter, &t, opts)
		return t, err
	}
}

// MetricSchema contains information about metrics in struct.
type MetricSchema struct {
	Metrics []MetricInfo
}

// WeaverYAML converts [MetricSchema] to Weaver YAML schema format.
func (s MetricSchema) WeaverYAML() []byte {
	return weaverYAMLFromSchemas(s.Metrics)
}

func weaverYAMLFromSchemas(schemas []MetricInfo) []byte {
	groups := make([]weaveryaml.Group, len(schemas))
	for i, sch := range schemas {
		groups[i] = sch.toGroup()
	}
	slices.SortFunc(groups, func(a, b weaveryaml.Group) int {
		return strings.Compare(a.ID, b.ID)
	})
	schema := weaveryaml.Schema{Groups: groups}
	return errors.Must(yaml.Marshal(schema))
}

// MetricInfo contains information about metric.
type MetricInfo struct {
	Type        string
	Instrument  string
	Name        string
	Description string
	Unit        string
}

func (m MetricInfo) toGroup() weaveryaml.Group {
	return weaveryaml.Group{
		ID:         fmt.Sprintf("metric.%s", m.Name),
		Type:       "metric",
		Instrument: m.Instrument,
		MetricName: m.Name,
		Brief:      m.Description,
		Unit:       m.Unit,
	}
}

// Schema returns information about metrics in given struct s.
func Schema(s any, opts InitOptions) (sch MetricSchema, err error) {
	opts.setDefaults()
	d := &descriptionCollectorMeter{
		Meter: metricnoop.NewMeterProvider().Meter("github.com/go-faster/autometric.Schema"),
	}
	if err := walkStruct(d, s, opts, func(reflect.Value, any) {}); err != nil {
		return sch, errors.Wrapf(err, "get schema from struct %T", s)
	}
	slices.SortFunc(d.infos, func(a, b MetricInfo) int {
		return cmp.Or(
			cmp.Compare(a.Type, b.Type),
			strings.Compare(a.Name, b.Name),
		)
	})
	return MetricSchema{Metrics: d.infos}, nil
}

type descriptionCollectorMeter struct {
	metric.Meter
	infos []MetricInfo
}

var _ metric.Meter = (*descriptionCollectorMeter)(nil)

func (d *descriptionCollectorMeter) addInfo(metricType, instrument, name, description, unit string) {
	d.infos = append(d.infos, MetricInfo{
		Type:        metricType,
		Instrument:  instrument,
		Name:        name,
		Description: description,
		Unit:        unit,
	})
}

// Float64Counter implements [metric.Meter].
func (d *descriptionCollectorMeter) Float64Counter(name string, options ...metric.Float64CounterOption) (metric.Float64Counter, error) {
	cfg := metric.NewFloat64CounterConfig(options...)
	d.addInfo("sum", "counter", name, cfg.Description(), cfg.Unit())
	return d.Meter.Float64Counter(name, options...)
}

// Float64Gauge implements [metric.Meter].
func (d *descriptionCollectorMeter) Float64Gauge(name string, options ...metric.Float64GaugeOption) (metric.Float64Gauge, error) {
	cfg := metric.NewFloat64GaugeConfig(options...)
	d.addInfo("gauge", "gauge", name, cfg.Description(), cfg.Unit())
	return d.Meter.Float64Gauge(name, options...)
}

// Float64Histogram implements [metric.Meter].
func (d *descriptionCollectorMeter) Float64Histogram(name string, options ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	cfg := metric.NewFloat64HistogramConfig(options...)
	d.addInfo("histogram", "histogram", name, cfg.Description(), cfg.Unit())
	return d.Meter.Float64Histogram(name, options...)
}

// Float64ObservableCounter implements [metric.Meter].
func (d *descriptionCollectorMeter) Float64ObservableCounter(name string, options ...metric.Float64ObservableCounterOption) (metric.Float64ObservableCounter, error) {
	cfg := metric.NewFloat64ObservableCounterConfig(options...)
	d.addInfo("sum", "counter", name, cfg.Description(), cfg.Unit())
	return d.Meter.Float64ObservableCounter(name, options...)
}

// Float64ObservableGauge implements [metric.Meter].
func (d *descriptionCollectorMeter) Float64ObservableGauge(name string, options ...metric.Float64ObservableGaugeOption) (metric.Float64ObservableGauge, error) {
	cfg := metric.NewFloat64ObservableGaugeConfig(options...)
	d.addInfo("gauge", "gauge", name, cfg.Description(), cfg.Unit())
	return d.Meter.Float64ObservableGauge(name, options...)
}

// Float64ObservableUpDownCounter implements [metric.Meter].
func (d *descriptionCollectorMeter) Float64ObservableUpDownCounter(name string, options ...metric.Float64ObservableUpDownCounterOption) (metric.Float64ObservableUpDownCounter, error) {
	cfg := metric.NewFloat64ObservableUpDownCounterConfig(options...)
	d.addInfo("sum", "updowncounter", name, cfg.Description(), cfg.Unit())
	return d.Meter.Float64ObservableUpDownCounter(name, options...)
}

// Float64UpDownCounter implements [metric.Meter].
func (d *descriptionCollectorMeter) Float64UpDownCounter(name string, options ...metric.Float64UpDownCounterOption) (metric.Float64UpDownCounter, error) {
	cfg := metric.NewFloat64UpDownCounterConfig(options...)
	d.addInfo("sum", "updowncounter", name, cfg.Description(), cfg.Unit())
	return d.Meter.Float64UpDownCounter(name, options...)
}

// Int64Counter implements [metric.Meter].
func (d *descriptionCollectorMeter) Int64Counter(name string, options ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	cfg := metric.NewInt64CounterConfig(options...)
	d.addInfo("sum", "counter", name, cfg.Description(), cfg.Unit())
	return d.Meter.Int64Counter(name, options...)
}

// Int64Gauge implements [metric.Meter].
func (d *descriptionCollectorMeter) Int64Gauge(name string, options ...metric.Int64GaugeOption) (metric.Int64Gauge, error) {
	cfg := metric.NewInt64GaugeConfig(options...)
	d.addInfo("gauge", "gauge", name, cfg.Description(), cfg.Unit())
	return d.Meter.Int64Gauge(name, options...)
}

// Int64Histogram implements [metric.Meter].
func (d *descriptionCollectorMeter) Int64Histogram(name string, options ...metric.Int64HistogramOption) (metric.Int64Histogram, error) {
	cfg := metric.NewInt64HistogramConfig(options...)
	d.addInfo("histogram", "histogram", name, cfg.Description(), cfg.Unit())
	return d.Meter.Int64Histogram(name, options...)
}

// Int64ObservableCounter implements [metric.Meter].
func (d *descriptionCollectorMeter) Int64ObservableCounter(name string, options ...metric.Int64ObservableCounterOption) (metric.Int64ObservableCounter, error) {
	cfg := metric.NewInt64ObservableCounterConfig(options...)
	d.addInfo("sum", "counter", name, cfg.Description(), cfg.Unit())
	return d.Meter.Int64ObservableCounter(name, options...)
}

// Int64ObservableGauge implements [metric.Meter].
func (d *descriptionCollectorMeter) Int64ObservableGauge(name string, options ...metric.Int64ObservableGaugeOption) (metric.Int64ObservableGauge, error) {
	cfg := metric.NewInt64ObservableGaugeConfig(options...)
	d.addInfo("gauge", "gauge", name, cfg.Description(), cfg.Unit())
	return d.Meter.Int64ObservableGauge(name, options...)
}

// Int64ObservableUpDownCounter implements [metric.Meter].
func (d *descriptionCollectorMeter) Int64ObservableUpDownCounter(name string, options ...metric.Int64ObservableUpDownCounterOption) (metric.Int64ObservableUpDownCounter, error) {
	cfg := metric.NewInt64ObservableUpDownCounterConfig(options...)
	d.addInfo("sum", "updowncounter", name, cfg.Description(), cfg.Unit())
	return d.Meter.Int64ObservableUpDownCounter(name, options...)
}

// Int64UpDownCounter implements [metric.Meter].
func (d *descriptionCollectorMeter) Int64UpDownCounter(name string, options ...metric.Int64UpDownCounterOption) (metric.Int64UpDownCounter, error) {
	cfg := metric.NewInt64UpDownCounterConfig(options...)
	d.addInfo("sum", "updowncounter", name, cfg.Description(), cfg.Unit())
	return d.Meter.Int64UpDownCounter(name, options...)
}

// RegisterCallback implements [metric.Meter].
func (d *descriptionCollectorMeter) RegisterCallback(f metric.Callback, instruments ...metric.Observable) (metric.Registration, error) {
	return d.Meter.RegisterCallback(f, instruments...)
}
