package autologs

import (
	"context"
	"io"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

// config contains configuration options for a LoggerProvider.
type config struct {
	res    *resource.Resource
	writer io.Writer
	lookup LookupExporter
}

// newConfig returns a config configured with options.
func newConfig(options []Option) config {
	conf := config{res: resource.Default()}
	for _, o := range options {
		conf = o.apply(conf)
	}
	return conf
}

// Option applies a configuration option value to a LoggerProvider.
type Option interface {
	apply(config) config
}

// optionFunc applies a set of options to a config.
type optionFunc func(config) config

// apply returns a config with option(s) applied.
func (o optionFunc) apply(conf config) config {
	return o(conf)
}

// WithResource associates a Resource with a LoggerProvider. This Resource
// represents the entity producing telemetry and is associated with all Meters
// the LoggerProvider will create.
//
// By default, if this Option is not used, the default Resource from the
// go.opentelemetry.io/otel/sdk/resource package will be used.
func WithResource(res *resource.Resource) Option {
	return optionFunc(func(conf config) config {
		conf.res = res
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
type LookupExporter func(ctx context.Context, name string) (sdklog.Exporter, bool, error)

// WithLookupExporter sets exporter lookup function.
func WithLookupExporter(lookup LookupExporter) Option {
	return optionFunc(func(conf config) config {
		conf.lookup = lookup
		return conf
	})
}
