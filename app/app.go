// Package app implements OTEL, prometheus, graceful shutdown and other common application features
// for go-faster projects.
package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/go-faster/errors"
	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"

	"github.com/go-faster/sdk/autologs"
	"github.com/go-faster/sdk/zctx"
)

const (
	exitCodeOk             = 0
	exitCodeApplicationErr = 1
	exitCodeWatchdog       = 1
)

const (
	shutdownTimeout = time.Second * 5
	watchdogTimeout = shutdownTimeout + time.Second*5
)

// Go runs f until interrupt.
func Go(f func(ctx context.Context, t *Telemetry) error, op ...Option) {
	Run(func(ctx context.Context, _ *zap.Logger, t *Telemetry) error {
		return f(ctx, t)
	}, op...)
}

// Run f until interrupt.
//
// If errors.Is(err, ctx.Err()) is valid for returned error, shutdown is considered graceful.
// Context is cancelled on SIGINT. After watchdogTimeout application is forcefully terminated
// with exitCodeWatchdog.
func Run(f func(ctx context.Context, lg *zap.Logger, m *Telemetry) error, op ...Option) {
	// Apply options.
	opts := options{
		zapConfig:  zap.NewProductionConfig(),
		zapTee:     true,
		ctx:        context.Background(),
		resourceFn: Resource,
	}
	if v, err := strconv.ParseBool(os.Getenv("OTEL_ZAP_TEE")); err == nil {
		// Override default.
		opts.zapTee = v
	}
	for _, o := range op {
		o.apply(&opts)
	}

	ctx, cancel := signal.NotifyContext(opts.ctx, os.Interrupt)
	defer cancel()

	// Setup logger.
	if s := os.Getenv("OTEL_LOG_LEVEL"); s != "" {
		var lvl zapcore.Level
		if err := lvl.UnmarshalText([]byte(s)); err != nil {
			panic(err)
		}
		opts.zapConfig.Level.SetLevel(lvl)
	}
	lg, err := opts.zapConfig.Build(opts.zapOptions...)
	if err != nil {
		panic(err)
	}
	defer func() { _ = lg.Sync() }()
	// Add logger to root context.
	ctx = zctx.Base(ctx, lg)

	lg.Info("Starting")
	res, err := opts.resourceFn(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to get resource: %v", err))
	}

	m, err := newTelemetry(ctx, lg.Named("metrics"), res, opts.meterOptions, opts.tracerOptions, opts.loggerOptions)
	if err != nil {
		panic(err)
	}

	// Setup logs.
	if ctx, err = autologs.Setup(ctx, m.LoggerProvider(), opts.zapTee); err != nil {
		panic(fmt.Sprintf("failed to setup logs: %v", err))
	}

	{
		// Automatically setting GOMAXPROCS.
		set := true // enabled by default
		if v, err := strconv.ParseBool(os.Getenv("AUTOMAXPROCS")); err == nil {
			set = v
		}
		minProcs := 1
		if v, err := strconv.Atoi(os.Getenv("AUTOMAXPROCS_MIN")); err == nil {
			minProcs = v
		}
		if set {
			if _, err := maxprocs.Set(
				maxprocs.Logger(lg.Sugar().Infof),
				maxprocs.Min(minProcs),
			); err != nil {
				lg.Warn("Failed to set GOMAXPROCS", zap.Error(err))
			}
		}
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() (rerr error) {
		defer lg.Info("Shutting down")
		defer func() {
			// Recovering panic to allow telemetry to flush.
			if ec := recover(); ec != nil {
				lg.Error("Panic",
					zap.String("panic", fmt.Sprintf("%v", ec)),
					zap.StackSkip("stack", 1),
				)
				rerr = fmt.Errorf("shutting down (panic): %v", ec)
			}
		}()
		if err := f(ctx, zctx.From(ctx), m); err != nil {
			if errors.Is(err, ctx.Err()) {
				// Parent context got cancelled, error is expected.
				lg.Debug("Graceful shutdown")
				return nil
			}
			return err
		}

		// Also shutting down metrics server to stop error group.
		cancel()

		return nil
	})
	g.Go(func() error {
		if err := m.run(ctx); err != nil {
			// Should already handle context cancellation gracefully.
			return errors.Wrap(err, "metrics")
		}
		return nil
	})

	go func() {
		// Guaranteed way to kill application.
		// Helps if f is stuck, e.g. deadlock during shutdown.
		<-ctx.Done()

		// Context is canceled, giving application time to shut down gracefully.

		lg.Info("Waiting for application shutdown")
		time.Sleep(watchdogTimeout)

		// Application is not shutting down gracefully, kill it.
		// This code should not be executed if f is already returned.

		lg.Warn("Graceful shutdown watchdog triggered: forcing shutdown")
		os.Exit(exitCodeWatchdog)
	}()

	if err := g.Wait(); err != nil {
		lg.Error("Failed", zap.Error(err))
		os.Exit(exitCodeApplicationErr)
	}

	lg.Info("Application stopped")
	os.Exit(exitCodeOk)
}
