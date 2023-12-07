package main

import (
	"context"
	"io"

	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/go-faster/sdk/app"
	"github.com/go-faster/sdk/autometer"
	"github.com/go-faster/sdk/autotracer"
)

func main() {
	app.Run(func(ctx context.Context, lg *zap.Logger, m *app.Metrics) error {
		lg.Info("Hello, world!")
		<-ctx.Done()
		lg.Info("Goodbye, world!")
		return nil
	},
		// Configure custom zap config.
		app.WithZapConfig(zap.NewDevelopmentConfig()),
		app.WithZapOptions(
			// Custom zap logger options.
			// E.g. hooks, custom core.
			zap.WrapCore(func(core zapcore.Core) zapcore.Core {
				return zapcore.NewTee(core)
			}),
		),

		// Redirect metrics and traces to /dev/null.
		app.WithMeterOptions(autometer.WithWriter(io.Discard)),
		app.WithTracerOptions(autotracer.WithWriter(io.Discard)),

		// Set base context. Background context is used by default.
		app.WithContext(context.Background()),

		// Allows to set custom resource.
		app.WithResource(func(ctx context.Context) (*resource.Resource, error) {
			return resource.Default(), nil
		}),
	)
}
