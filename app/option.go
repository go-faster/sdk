package app

import "go.uber.org/zap"

type options struct {
	cfg zap.Config
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
