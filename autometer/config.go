package autometer

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/sdk/resource"
)

// config contains configuration options for a MeterProvider.
type config struct {
	res *resource.Resource

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
