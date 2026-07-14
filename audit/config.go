package audit

import (
	"time"

	"go.opentelemetry.io/otel/sdk/resource"
)

// config contains configuration options for a Recorder.
type config struct {
	exporters   []Exporter
	redactor    Redactor
	resource    *resource.Resource
	service     string
	component   string
	environment string
	version     string
	clock       func() time.Time
	idGenerator func() string
	traceEnrich bool
}

// newConfig returns a config configured with options. After folding options,
// provenance fields still empty are filled from the [resource.Resource] (if
// set), so explicit [WithService]/[WithComponent]/... win over resource attrs.
func newConfig(options []Option) config {
	conf := config{
		redactor:    NoRedact(),
		clock:       defaultClock,
		idGenerator: defaultIDGenerator,
	}
	for _, o := range options {
		conf = o.apply(conf)
	}
	if conf.resource != nil {
		rs := fromResource(conf.resource)
		if conf.service == "" {
			conf.service = rs.service
		}
		if conf.component == "" {
			conf.component = rs.component
		}
		if conf.version == "" {
			conf.version = rs.version
		}
		if conf.environment == "" {
			conf.environment = rs.environment
		}
	}
	return conf
}

// Option applies a configuration option value to a Recorder.
type Option interface {
	apply(config) config
}

// optionFunc applies a set of options to a config.
type optionFunc func(config) config

// apply returns a config with option(s) applied.
func (o optionFunc) apply(conf config) config {
	return o(conf)
}

// WithExporter adds an exporter.
func WithExporter(exporter Exporter) Option {
	return optionFunc(func(conf config) config {
		if exporter != nil {
			conf.exporters = append(conf.exporters, exporter)
		}
		return conf
	})
}

// WithExporters adds exporters.
func WithExporters(exporters ...Exporter) Option {
	return optionFunc(func(conf config) config {
		for _, exporter := range exporters {
			if exporter != nil {
				conf.exporters = append(conf.exporters, exporter)
			}
		}
		return conf
	})
}

// WithRedactor sets the redactor.
func WithRedactor(redactor Redactor) Option {
	return optionFunc(func(conf config) config {
		if redactor != nil {
			conf.redactor = redactor
		}
		return conf
	})
}

// WithResource sets the OpenTelemetry [resource.Resource] used to populate
// provenance defaults (Service, Component, Version, Environment) from
// OpenTelemetry semantic-convention attributes when the corresponding Event
// field is empty at emit time.
//
// Explicit [WithService], [WithComponent], [WithVersion], and [WithEnvironment]
// options take precedence over resource attributes.
func WithResource(r *resource.Resource) Option {
	return optionFunc(func(conf config) config {
		conf.resource = r
		return conf
	})
}

// WithService sets default service field.
func WithService(service string) Option {
	return optionFunc(func(conf config) config { conf.service = service; return conf })
}

// WithComponent sets default component field.
func WithComponent(component string) Option {
	return optionFunc(func(conf config) config { conf.component = component; return conf })
}

// WithEnvironment sets default environment field.
func WithEnvironment(environment string) Option {
	return optionFunc(func(conf config) config { conf.environment = environment; return conf })
}

// WithVersion sets default version field.
func WithVersion(version string) Option {
	return optionFunc(func(conf config) config { conf.version = version; return conf })
}

// WithClock sets clock used by Emit.
func WithClock(clock func() time.Time) Option {
	return optionFunc(func(conf config) config {
		if clock != nil {
			conf.clock = clock
		}
		return conf
	})
}

// WithIDGenerator sets ID generator used by Emit.
func WithIDGenerator(fn func() string) Option {
	return optionFunc(func(conf config) config {
		if fn != nil {
			conf.idGenerator = fn
		}
		return conf
	})
}

// WithTraceEnrichment enables enrichment of [Event.TraceID] and [Event.SpanID]
// from the active OpenTelemetry span context in [Recorder.Record]'s ctx.
//
// When set, Record stamps TraceID/SpanID from trace.SpanContextFromContext(ctx)
// if and only if a span is valid and the corresponding Event field is empty.
// This is opt-in: trace_id/span_id are OpenTelemetry-reserved fields and are
// never synthesized from CorrelationID or other IDs.
func WithTraceEnrichment() Option {
	return optionFunc(func(conf config) config {
		conf.traceEnrich = true
		return conf
	})
}
