package main

import (
	"context"
	"io"

	"go.uber.org/zap"

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

		// Redirect metrics and traces to /dev/null.
		app.WithMeterOptions(autometer.WithWriter(io.Discard)),
		app.WithTracerOptions(autotracer.WithWriter(io.Discard)),
	)
}
