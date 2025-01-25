package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-faster/errors"
	"github.com/go-logr/zapr"
	otelpyroscope "github.com/grafana/otel-profiling-go"
	promClient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/go-faster/sdk/autologs"
	"github.com/go-faster/sdk/autometer"
	"github.com/go-faster/sdk/autopyro"
	"github.com/go-faster/sdk/autotracer"
)

type httpEndpoint struct {
	srv      *http.Server
	mux      *http.ServeMux
	services []string
	addr     string
}

// Deprecated: use Telemetry.
type Metrics = Telemetry

// Telemetry implement common basic metrics and infrastructure to it.
type Telemetry struct {
	lg *zap.Logger

	prom *promClient.Registry
	http []httpEndpoint

	tracerProvider trace.TracerProvider
	meterProvider  metric.MeterProvider
	loggerProvider log.LoggerProvider

	resource   *resource.Resource
	propagator propagation.TextMapPropagator

	shutdowns []shutdown
}

func (m *Telemetry) registerShutdown(name string, fn func(ctx context.Context) error) {
	m.shutdowns = append(m.shutdowns, shutdown{name: name, fn: fn})
}

type shutdown struct {
	name string
	fn   func(ctx context.Context) error
}

func (m *Telemetry) String() string {
	return "metrics"
}

func (m *Telemetry) run(ctx context.Context) error {
	defer m.lg.Debug("Stopped metrics")
	wg, ctx := errgroup.WithContext(ctx)

	for i := range m.http {
		e := m.http[i]
		wg.Go(func() error {
			m.lg.Info("Starting http server",
				zap.Strings("services", e.services),
			)
			if err := e.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			m.lg.Debug("Metrics server gracefully stopped")
			return nil
		})
	}
	wg.Go(func() error {
		// Wait until g ctx canceled, then try to shut down server.
		<-ctx.Done()

		m.lg.Debug("Shutting down metrics")
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		// Not returning error, just reporting to log.
		m.shutdown(ctx)

		return nil
	})

	return wg.Wait()
}

func (m *Telemetry) shutdown(ctx context.Context) {
	var wg sync.WaitGroup

	// Launch shutdowns in parallel.
	wg.Add(len(m.shutdowns))

	var shutdowns []string
	for _, s := range m.shutdowns {
		var (
			f = s.fn
			n = s.name
		)
		shutdowns = append(shutdowns, n)
		go func() {
			defer wg.Done()
			if err := f(ctx); err != nil {
				m.lg.Error("Failed to shutdown", zap.Error(err), zap.String("name", n))
			}
		}()
	}

	// Wait for all shutdowns to finish.
	m.lg.Info("Waiting for shutdowns", zap.Strings("shutdowns", shutdowns))
	wg.Wait()
}

func (m *Telemetry) MeterProvider() metric.MeterProvider {
	if m.meterProvider == nil {
		return otel.GetMeterProvider()
	}
	return m.meterProvider
}

func (m *Telemetry) TracerProvider() trace.TracerProvider {
	if m.tracerProvider == nil {
		return otel.GetTracerProvider()
	}
	return m.tracerProvider
}

func (m *Telemetry) LoggerProvider() log.LoggerProvider {
	if m.loggerProvider == nil {
		return noop.NewLoggerProvider()
	}
	return m.loggerProvider
}

func (m *Telemetry) TextMapPropagator() propagation.TextMapPropagator {
	return m.propagator
}

func prometheusAddr() string {
	host := "localhost"
	port := "9464"
	if v := os.Getenv("OTEL_EXPORTER_PROMETHEUS_HOST"); v != "" {
		host = v
	}
	if v := os.Getenv("OTEL_EXPORTER_PROMETHEUS_PORT"); v != "" {
		port = v
	}
	return net.JoinHostPort(host, port)
}

type zapErrorHandler struct {
	lg *zap.Logger
}

func (z zapErrorHandler) Handle(err error) {
	z.lg.Error("Error", zap.Error(err))
}

func newTelemetry(
	ctx context.Context,
	lg *zap.Logger,
	res *resource.Resource,
	meterOptions []autometer.Option,
	tracerOptions []autotracer.Option,
	logsOptions []autologs.Option,
) (*Telemetry, error) {
	{
		// Setup global OTEL logger and error handler.
		logger := lg.Named("otel")
		otel.SetLogger(zapr.NewLogger(logger))
		otel.SetErrorHandler(zapErrorHandler{lg: logger})
	}
	m := &Telemetry{
		lg:       lg,
		resource: res,
	}
	{
		provider, stop, err := autologs.NewLoggerProvider(ctx,
			include(logsOptions,
				autologs.WithResource(res),
			)...,
		)
		if err != nil {
			return nil, errors.Wrap(err, "logger provider")
		}
		m.loggerProvider = provider
		m.registerShutdown("logger", stop)
	}
	{
		provider, stop, err := autotracer.NewTracerProvider(ctx,
			include(tracerOptions,
				autotracer.WithResource(res),
			)...,
		)
		if err != nil {
			return nil, errors.Wrap(err, "tracer provider")
		}
		m.tracerProvider = provider
		m.registerShutdown("tracer", stop)
	}
	{
		provider, stop, err := autometer.NewMeterProvider(ctx,
			include(meterOptions,
				autometer.WithResource(res),
				autometer.WithOnPrometheusRegistry(func(reg *promClient.Registry) {
					m.prom = reg
				}),
			)...,
		)
		if err != nil {
			return nil, errors.Wrap(err, "meter provider")
		}
		m.meterProvider = provider
		m.registerShutdown("meter", stop)
	}

	// Automatically composited from the OTEL_PROPAGATORS environment variable.
	m.propagator = autoprop.NewTextMapPropagator()

	// Setting up go runtime metrics.
	if err := runtime.Start(
		runtime.WithMeterProvider(m.MeterProvider()),
		runtime.WithMinimumReadMemStatsInterval(time.Second), // export as env?
	); err != nil {
		return nil, errors.Wrap(err, "runtime metrics")
	}

	// Setup pyroscope.
	if autopyro.Enabled() {
		stop, err := autopyro.Setup(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "pyroscope")
		}
		m.registerShutdown("pyroscope", stop)
		// Setup pyroscope tracing integration.
		// See https://github.com/grafana/otel-profiling-go
		m.tracerProvider = otelpyroscope.NewTracerProvider(m.tracerProvider)
	}

	// Register global OTEL providers.
	otel.SetMeterProvider(m.MeterProvider())
	otel.SetTracerProvider(m.TracerProvider())
	otel.SetTextMapPropagator(m.TextMapPropagator())

	// Initialize and register HTTP servers if required.
	//
	// Adding prometheus.
	if m.prom != nil {
		promAddr := prometheusAddr()
		if v := os.Getenv("METRICS_ADDR"); v != "" {
			promAddr = v
		}
		mux := http.NewServeMux()
		e := httpEndpoint{
			srv:      &http.Server{Addr: promAddr, Handler: mux},
			services: []string{"prometheus"},
			addr:     promAddr,
			mux:      mux,
		}
		mux.Handle("/metrics",
			promhttp.HandlerFor(m.prom, promhttp.HandlerOpts{}),
		)
		m.http = append(m.http, e)
	}
	// Adding pprof.
	if v := os.Getenv("PPROF_ADDR"); v != "" {
		const serviceName = "pprof"
		// Search for existing endpoint.
		var he httpEndpoint
		for i, e := range m.http {
			if e.addr != v {
				continue
			}
			// Using existing endpoint
			he = e
			he.services = append(he.services, serviceName)
			m.http[i] = he
		}
		if he.srv == nil {
			// Creating new endpoint.
			mux := http.NewServeMux()
			he = httpEndpoint{
				srv:      &http.Server{Addr: v, Handler: mux},
				addr:     v,
				mux:      mux,
				services: []string{serviceName},
			}
			m.http = append(m.http, he)
		}
		m.registerProfiler(he.mux)
	}
	fields := []zap.Field{
		zap.Stringer("otel.resource", res),
	}
	for _, e := range m.http {
		for _, s := range e.services {
			fields = append(fields, zap.String("http."+s, e.addr))
		}
		name := fmt.Sprintf("http %v", e.services)
		m.registerShutdown(name, e.srv.Shutdown)
	}
	lg.Info("Metrics initialized", fields...)
	return m, nil
}
