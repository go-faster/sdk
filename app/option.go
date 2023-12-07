package app

import (
	"context"

	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap"

	"github.com/go-faster/sdk/autometer"
	"github.com/go-faster/sdk/autotracer"
)

type options struct {
	zapConfig  zap.Config
	zapOptions []zap.Option
	ctx        context.Context

	meterOptions  []autometer.Option
	tracerOptions []autotracer.Option
	resourceFn    func(ctx context.Context) (*resource.Resource, error)
}

type optionFunc func(*options)

func (f optionFunc) apply(o *options) {
	f(o)
}

// Option is a functional option for the application.
type Option interface {
	apply(o *options)
}

// WithZapConfig sets the default zap config for the application.
func WithZapConfig(cfg zap.Config) Option {
	return optionFunc(func(o *options) {
		o.zapConfig = cfg
	})
}

// WithZapOptions sets additional zap logger options for the application.
func WithZapOptions(opts ...zap.Option) Option {
	return optionFunc(func(o *options) {
		o.zapOptions = opts
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

// WithContext sets the base context for the application. Background context is used by default.
func WithContext(ctx context.Context) Option {
	return optionFunc(func(o *options) {
		o.ctx = ctx
	})
}

// WithResource sets the function that will be called to retrieve telemetry resource for application.
//
// Defaults to [Resource] function.
func WithResource(fn func(ctx context.Context) (*resource.Resource, error)) Option {
	return optionFunc(func(o *options) {
		o.resourceFn = fn
	})
}
