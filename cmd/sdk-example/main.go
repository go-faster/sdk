package main

import (
	"context"

	"go.uber.org/zap"

	"github.com/go-faster/sdk/app"
)

func main() {
	app.Run(func(ctx context.Context, lg *zap.Logger, m *app.Metrics) error {
		lg.Info("Hello, world!")
		<-ctx.Done()
		lg.Info("Goodbye, world!")
		return nil
	})
}
