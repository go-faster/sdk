package autometer

import (
	"context"
	"io"

	"github.com/prometheus/client_golang/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

// config contains configuration options for a MeterProvider.
type config struct {
	res    *resource.Resource
	writer io.Writer
	lookup LookupExporter

	prom         prometheus.Registerer
	promCallback func(reg *prometheus.Registry)
}

// newConfig returns a config configured with options.
func newConfig(options []Option) config {
	conf := config{res: resource.Default()}
	for _, o := range options {
		conf = o.apply(conf)
	}
	return conf
}

// Option applies a configuration option value to a MeterProvider.
type Option interface {
	apply(config) config
}

// optionFunc applies a set of options to a config.
type optionFunc func(config) config

// apply returns a config with option(s) applied.
func (o optionFunc) apply(conf config) config {
	return o(conf)
}

// WithResource associates a Resource with a MeterProvider. This Resource
// represents the entity producing telemetry and is associated with all Meters
// the MeterProvider will create.
//
// By default, if this Option is not used, the default Resource from the
// go.opentelemetry.io/otel/sdk/resource package will be used.
func WithResource(res *resource.Resource) Option {
	return optionFunc(func(conf config) config {
		conf.res = res
		return conf
	})
}

func WithPrometheusRegisterer(reg prometheus.Registerer) Option {
	return optionFunc(func(conf config) config {
		conf.prom = reg
		return conf
	})
}

func WithOnPrometheusRegistry(f func(reg *prometheus.Registry)) Option {
	return optionFunc(func(conf config) config {
		conf.promCallback = f
		return conf
	})
}

// WithWriter sets writer for the stderr, stdout exporters.
func WithWriter(out io.Writer) Option {
	return optionFunc(func(conf config) config {
		conf.writer = out
		return conf
	})
}

// LookupExporter creates exporter by name.
type LookupExporter func(ctx context.Context, name string) (sdkmetric.Reader, bool, error)

// WithLookupExporter sets exporter lookup function.
func WithLookupExporter(lookup LookupExporter) Option {
	return optionFunc(func(conf config) config {
		conf.lookup = lookup
		return conf
	})
}
