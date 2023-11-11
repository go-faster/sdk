package app

import (
	"context"

	"go.uber.org/zap"

	"github.com/go-faster/sdk/autometer"
	"github.com/go-faster/sdk/autotracer"
)

type options struct {
	cfg zap.Config
	ctx context.Context

	meterOptions  []autometer.Option
	tracerOptions []autotracer.Option
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
		o.cfg = cfg
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
