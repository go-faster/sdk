// Package autometric contains a simple reflect-based OpenTelemetry metric initializer.
package autometric

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/go-faster/errors"
	"go.opentelemetry.io/otel/metric"
)

var (
	int64CounterType                 = reflect.TypeFor[metric.Int64Counter]()
	int64UpDownCounterType           = reflect.TypeFor[metric.Int64UpDownCounter]()
	int64HistogramType               = reflect.TypeFor[metric.Int64Histogram]()
	int64GaugeType                   = reflect.TypeFor[metric.Int64Gauge]()
	int64ObservableCounterType       = reflect.TypeFor[metric.Int64ObservableCounter]()
	int64ObservableUpDownCounterType = reflect.TypeFor[metric.Int64ObservableUpDownCounter]()
	int64ObservableGaugeType         = reflect.TypeFor[metric.Int64ObservableGauge]()
)

var (
	float64CounterType                 = reflect.TypeFor[metric.Float64Counter]()
	float64UpDownCounterType           = reflect.TypeFor[metric.Float64UpDownCounter]()
	float64HistogramType               = reflect.TypeFor[metric.Float64Histogram]()
	float64GaugeType                   = reflect.TypeFor[metric.Float64Gauge]()
	float64ObservableCounterType       = reflect.TypeFor[metric.Float64ObservableCounter]()
	float64ObservableUpDownCounterType = reflect.TypeFor[metric.Float64ObservableUpDownCounter]()
	float64ObservableGaugeType         = reflect.TypeFor[metric.Float64ObservableGauge]()
)

// InitOptions defines options for [Init].
type InitOptions struct {
	// Prefix defines common prefix for all metrics.
	Prefix string
	// FieldName returns name for given field.
	FieldName func(prefix string, sf reflect.StructField) string
}

func (opts *InitOptions) setDefaults() {
	if opts.FieldName == nil {
		opts.FieldName = fieldName
	}
}

func fieldName(prefix string, sf reflect.StructField) string {
	name := snakeCase(sf.Name)
	if tag, ok := sf.Tag.Lookup("name"); ok {
		name = tag
	}
	return prefix + name
}

// Init initialize metrics in given struct s using given meter.
func Init(m metric.Meter, s any, opts InitOptions) error {
	return walkStruct(m, s, opts, func(field reflect.Value, mt any) {
		if !field.CanSet() {
			return
		}
		field.Set(reflect.ValueOf(mt))
	})
}

func walkStruct(m metric.Meter, s any, opts InitOptions, fn func(field reflect.Value, mt any)) error {
	opts.setDefaults()

	ptr := reflect.ValueOf(s)
	if ptr.Kind() != reflect.Pointer || ptr.Elem().Kind() != reflect.Struct {
		if ptr.Kind() == reflect.Pointer && ptr.IsNil() {
			return errors.Errorf("a pointer-to-struct expected, got (%T)(nil)", s)
		}
		return errors.Errorf("a pointer-to-struct expected, got %T", s)
	}

	struct_ := ptr.Elem()
	structType := struct_.Type()
	for i := 0; i < struct_.NumField(); i++ {
		fieldType := structType.Field(i)
		if fieldType.Anonymous || !fieldType.IsExported() {
			continue
		}
		if n, ok := fieldType.Tag.Lookup("autometric"); ok && n == "-" {
			continue
		}
		field := struct_.Field(i)

		mt, err := makeField(m, fieldType, opts)
		if err != nil {
			return errors.Wrapf(err, "field (%s).%s", structType, fieldType.Name)
		}
		fn(field, mt)
	}

	return nil
}

func makeField(m metric.Meter, sf reflect.StructField, opts InitOptions) (any, error) {
	var (
		name       = opts.FieldName(opts.Prefix, sf)
		unit       = sf.Tag.Get("unit")
		desc       = sf.Tag.Get("description")
		boundaries []float64
	)
	if b, ok := sf.Tag.Lookup("boundaries"); ok {
		switch ftyp := sf.Type; ftyp {
		case int64HistogramType, float64HistogramType:
		default:
			return nil, errors.Errorf("boundaries tag should be used only on histogram metrics: got %v", ftyp)
		}
		for val := range strings.SplitSeq(b, ",") {
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, errors.Wrap(err, "parse boundaries")
			}
			boundaries = append(boundaries, f)
		}
	}

	switch ftyp := sf.Type; ftyp {
	case int64CounterType:
		return m.Int64Counter(name,
			metric.WithUnit(unit),
			metric.WithDescription(desc),
		)
	case int64UpDownCounterType:
		return m.Int64UpDownCounter(name,
			metric.WithUnit(unit),
			metric.WithDescription(desc),
		)
	case int64HistogramType:
		return m.Int64Histogram(name,
			metric.WithUnit(unit),
			metric.WithDescription(desc),
			metric.WithExplicitBucketBoundaries(boundaries...),
		)
	case int64GaugeType:
		return m.Int64Gauge(name,
			metric.WithUnit(unit),
			metric.WithDescription(desc),
		)
	case int64ObservableCounterType,
		int64ObservableUpDownCounterType,
		int64ObservableGaugeType:
		return nil, errors.New("observables are not supported")

	case float64CounterType:
		return m.Float64Counter(name,
			metric.WithUnit(unit),
			metric.WithDescription(desc),
		)
	case float64UpDownCounterType:
		return m.Float64UpDownCounter(name,
			metric.WithUnit(unit),
			metric.WithDescription(desc),
		)
	case float64HistogramType:
		return m.Float64Histogram(name,
			metric.WithUnit(unit),
			metric.WithDescription(desc),
			metric.WithExplicitBucketBoundaries(boundaries...),
		)
	case float64GaugeType:
		return m.Float64Gauge(name,
			metric.WithUnit(unit),
			metric.WithDescription(desc),
		)
	case float64ObservableCounterType,
		float64ObservableUpDownCounterType,
		float64ObservableGaugeType:
		return nil, errors.New("observables are not supported")
	default:
		return nil, errors.Errorf("unexpected type %v", ftyp)
	}
}
