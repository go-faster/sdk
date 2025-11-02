package app

import (
	"context"

	"github.com/go-faster/sdk/internal/zapencoder"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.uber.org/zap"

	"github.com/go-faster/sdk/autologs"
	"github.com/go-faster/sdk/autometer"
	"github.com/go-faster/sdk/autotracer"
)

type options struct {
	zapConfig       zap.Config
	zapCustomConfig bool
	zapOptions      []zap.Option
	zapTee          bool
	otelZap         bool
	ctx             context.Context

	meterOptions    []autometer.Option
	tracerOptions   []autotracer.Option
	loggerOptions   []autologs.Option
	resourceOptions []resource.Option
	resourceFn      func(ctx context.Context) (*resource.Resource, error)
}

func (o *options) buildLogger() *zap.Logger {
	if !o.zapCustomConfig {
		// HACK: to use custom encoding for Any("ctx', ctx) fields
		// we need to use custom zap.Config and custom encoder.
		cfg := zapencoder.Config(defaultZapConfig())
		lg, err := cfg.Build(o.zapOptions...)
		if err != nil {
			panic("failed to build zap logger: " + err.Error())
		}
		return lg
	}

	lg, err := o.zapConfig.Build(o.zapOptions...)
	if err != nil {
		panic("failed to build zap logger: " + err.Error())
	}

	return lg
}

type optionFunc func(*options)

func (f optionFunc) apply(o *options) {
	f(o)
}

// Option is a functional option for the application.
type Option interface {
	apply(o *options)
}

// WithZapTee sets option to tee zap logs to stderr.
func WithZapTee(teeToStderr bool) Option {
	return optionFunc(func(o *options) {
		o.zapTee = teeToStderr
	})
}

// WithZapConfig sets the default zap config for the application.
func WithZapConfig(cfg zap.Config) Option {
	return optionFunc(func(o *options) {
		o.zapConfig = cfg
		o.zapCustomConfig = true
	})
}

// WithZapOptions sets additional zap logger options for the application.
func WithZapOptions(opts ...zap.Option) Option {
	return optionFunc(func(o *options) {
		o.zapOptions = opts
	})
}

// WithZapOpenTelemetry enabels OpenTelemetry mode for zap.
// See [zctx.WithOpenTelemetryZap].
func WithZapOpenTelemetry() Option {
	return optionFunc(func(o *options) {
		o.otelZap = true
	})
}

// WithMeterOptions sets the default autometer options for the application.
func WithMeterOptions(opts ...autometer.Option) Option {
	return optionFunc(func(o *options) {
		o.meterOptions = opts
	})
}

// WithTracerOptions sets the default autotracer options for the application.
func WithTracerOptions(opts ...autotracer.Option) Option {
	return optionFunc(func(o *options) {
		o.tracerOptions = opts
	})
}

// WithResourceOptions sets the default resource options.
//
// Use before [WithResource] or [WithServiceName] to override default resource options.
func WithResourceOptions(opts ...resource.Option) Option {
	return optionFunc(func(o *options) {
		o.resourceOptions = opts
	})
}

// WithServiceName sets the default service name for the application.
func WithServiceName(name string) Option {
	return optionFunc(func(o *options) {
		o.resourceOptions = append(o.resourceOptions, resource.WithAttributes(semconv.ServiceName(name)))
	})
}

// WithServiceNamespace sets the default service namespace for the application.
func WithServiceNamespace(namespace string) Option {
	return optionFunc(func(o *options) {
		o.resourceOptions = append(o.resourceOptions, resource.WithAttributes(semconv.ServiceNamespace(namespace)))
	})
}

// WithContext sets the base context for the application. Background context is used by default.
func WithContext(ctx context.Context) Option {
	return optionFunc(func(o *options) {
		o.ctx = ctx
	})
}

// WithResource sets the function that will be called to retrieve telemetry resource for application.
//
// Defaults to function that enables most common resource detectors.
func WithResource(fn func(ctx context.Context) (*resource.Resource, error)) Option {
	return optionFunc(func(o *options) {
		o.resourceFn = fn
	})
}
