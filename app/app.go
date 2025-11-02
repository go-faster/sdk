// Package app implements OTEL, prometheus, graceful shutdown and other common application features
// for go-faster projects.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/KimMachineGun/automemlimit/memlimit"
	"github.com/go-faster/errors"
	"github.com/go-faster/sdk/internal/zapencoder"
	slogzap "github.com/samber/slog-zap/v2"
	"go.opentelemetry.io/otel/sdk/resource"
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

var _registerZapEncoder sync.Once

func defaultZapConfig() zap.Config {
	cfg := zap.NewProductionConfig()
	const encoderName = "github.com/go-faster/sdk/zapencoder.JSON"
	_registerZapEncoder.Do(func() {
		_ = zap.RegisterEncoder(encoderName, func(config zapcore.EncoderConfig) (zapcore.Encoder, error) {
			return zapencoder.New(config), nil
		})
	})
	cfg.Encoding = encoderName
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return cfg
}

// Run f until interrupt.
//
// If errors.Is(err, ctx.Err()) is valid for returned error, shutdown is considered graceful.
// Context is cancelled on SIGINT. After watchdogTimeout application is forcefully terminated
// with exitCodeWatchdog.
func Run(f func(ctx context.Context, lg *zap.Logger, t *Telemetry) error, op ...Option) {
	// Apply options.
	opts := options{
		zapConfig: defaultZapConfig(),
		zapTee:    true,
		ctx:       context.Background(),
		resourceOptions: []resource.Option{
			resource.WithProcessRuntimeDescription(),
			resource.WithProcessRuntimeVersion(),
			resource.WithProcessRuntimeName(),
			resource.WithOS(),
			resource.WithFromEnv(),
			resource.WithTelemetrySDK(),
			resource.WithHost(),
			resource.WithProcess(),
		},
	}
	opts.resourceFn = func(ctx context.Context) (*resource.Resource, error) {
		r, err := resource.New(ctx, opts.resourceOptions...)
		if err != nil {
			return nil, errors.Wrap(err, "new")
		}
		return resource.Merge(resource.Default(), r)
	}
	if v, err := strconv.ParseBool(os.Getenv("OTEL_ZAP_TEE")); err == nil {
		// Override default.
		opts.zapTee = v
	}
	for _, o := range op {
		o.apply(&opts)
	}

	ctx := opts.ctx
	if opts.otelZap {
		ctx = zctx.WithOpenTelemetryZap(ctx)
	}
	ctx, baseCtxCancel := context.WithCancel(ctx)
	defer baseCtxCancel()

	// Setup logger.
	if s := os.Getenv("OTEL_LOG_LEVEL"); s != "" {
		var lvl zapcore.Level
		if err := lvl.UnmarshalText([]byte(s)); err != nil {
			panic(err)
		}
		opts.zapConfig.Level.SetLevel(lvl)
	}
	lg := opts.buildLogger()
	defer func() { _ = lg.Sync() }()
	// Add logger to root context.
	ctx = zctx.Base(ctx, lg)

	// Explicit context for graceful shutdown.
	shutdownCtx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	lg.Info("Starting")
	res, err := opts.resourceFn(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to get resource: %v", err))
	}

	m, err := newTelemetry(
		ctx, shutdownCtx,
		lg.Named("metrics"),
		res,
		opts.meterOptions, opts.tracerOptions, opts.loggerOptions,
	)
	if err != nil {
		panic(err)
	}

	// Setup logs.
	if ctx, err = autologs.Setup(ctx, m.LoggerProvider(), opts.zapTee); err != nil {
		panic(fmt.Sprintf("failed to setup logs: %v", err))
	}

	shutdownCtx = zctx.Base(shutdownCtx, zctx.From(ctx))
	m.shutdownContext = shutdownCtx
	m.baseContext = ctx

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
	{
		// Automatically set GOMEMLIMIT.
		// https://github.com/KimMachineGun/automemlimit
		// https://tip.golang.org/doc/gc-guide#Memory_limit
		logger := slog.New(slogzap.Option{Level: slog.LevelDebug, Logger: lg}.NewZapHandler())
		if _, err := memlimit.SetGoMemLimitWithOpts(memlimit.WithLogger(logger)); err != nil {
			lg.Warn("Failed to set memory limit", zap.Error(err))
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
		m.baseContext = ctx
		if err := f(m.shutdownContext, zctx.From(ctx), m); err != nil {
			if errors.Is(err, ctx.Err()) {
				// Parent context got cancelled, error is expected.
				// TODO(ernado): check for shutdownCtx instead.
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
		<-shutdownCtx.Done()
		lg.Info("Shutdown triggered. Waiting for graceful shutdown")
		time.Sleep(shutdownTimeout)
		baseCtxCancel()

		// Context is canceled, giving application time to shut down gracefully.

		lg.Info("Base context cancelled. Forcing shutdown")
		time.Sleep(watchdogTimeout)

		// Application is not shutting down gracefully, kill it.
		// This code should not be executed if f is already returned.

		lg.Warn("Graceful shutdown watchdog triggered: forcing hard shutdown")
		os.Exit(exitCodeWatchdog)
	}()

	if err := g.Wait(); err != nil {
		lg.Error("Failed", zap.Error(err))
		os.Exit(exitCodeApplicationErr)
	}

	lg.Info("Application stopped")
	os.Exit(exitCodeOk)
}
