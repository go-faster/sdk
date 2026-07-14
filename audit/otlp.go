package audit

import (
	"context"
	"strconv"

	"go.opentelemetry.io/otel/log"

	"github.com/go-faster/sdk/autologs"
)

// OTLPExporter exports audit events through OpenTelemetry Logs.
type OTLPExporter struct {
	logger log.Logger
	stop   autologs.ShutdownFunc
	owned  bool
}

type otlpExporterConfig struct {
	provider log.LoggerProvider
	options  []autologs.Option
}

// OTLPExporterOption configures OTLPExporter.
type OTLPExporterOption interface {
	applyOTLP(otlpExporterConfig) otlpExporterConfig
}

type otlpExporterOptionFunc func(otlpExporterConfig) otlpExporterConfig

func (f otlpExporterOptionFunc) applyOTLP(conf otlpExporterConfig) otlpExporterConfig { return f(conf) }

// WithLoggerProvider sets a shared LoggerProvider.
func WithLoggerProvider(p log.LoggerProvider) OTLPExporterOption {
	return otlpExporterOptionFunc(func(conf otlpExporterConfig) otlpExporterConfig {
		conf.provider = p
		return conf
	})
}

// WithOTLPOptions sets options used when creating an owned LoggerProvider.
func WithOTLPOptions(options []autologs.Option) OTLPExporterOption {
	return otlpExporterOptionFunc(func(conf otlpExporterConfig) otlpExporterConfig {
		conf.options = options
		return conf
	})
}

// NewOTLPExporter creates an OTLPExporter.
func NewOTLPExporter(ctx context.Context, opts ...OTLPExporterOption) (*OTLPExporter, error) {
	var cfg otlpExporterConfig
	for _, o := range opts {
		cfg = o.applyOTLP(cfg)
	}
	if cfg.provider != nil {
		return &OTLPExporter{logger: cfg.provider.Logger("audit")}, nil
	}
	provider, stop, err := autologs.NewLoggerProvider(ctx, cfg.options...)
	if err != nil {
		return nil, err
	}
	return &OTLPExporter{logger: provider.Logger("audit"), stop: stop, owned: true}, nil
}

// Export implements Exporter.
func (o *OTLPExporter) Export(ctx context.Context, events []Event) error {
	for _, event := range events {
		r := eventToLogRecord(event)
		o.logger.Emit(ctx, r)
	}
	return nil
}

// Close implements Exporter.
func (o *OTLPExporter) Close(ctx context.Context) error {
	if o.owned && o.stop != nil {
		return o.stop(ctx)
	}
	return nil
}

// Name implements Exporter.
func (o *OTLPExporter) Name() string { return "otlp" }

func eventToLogRecord(e Event) log.Record {
	var r log.Record
	r.SetTimestamp(e.Time)
	r.SetSeverityText(strconv.Itoa(int(e.Severity)))
	r.SetSeverity(severityToOTelSeverity(e.Severity))
	r.SetBody(log.StringValue(e.Message))
	r.AddAttributes(eventToLogAttributes(e)...)
	return r
}

func eventToLogAttributes(e Event) []log.KeyValue {
	attrs := []log.KeyValue{
		log.String("event.id", e.ID),
		log.String("event.type", string(e.Type)),
		log.String("event.action", e.Action),
		log.String("event.result", string(e.Result)),
		log.Int("event.severity", int(e.Severity)),
		log.String("event.reason", e.Reason),
		log.String("actor.id", e.ActorID),
		log.String("actor.type", string(e.ActorType)),
		log.String("actor.session_id", e.SessionID),
		log.String("actor.auth_method", e.AuthMethod),
		log.String("target.type", e.TargetType),
		log.String("target.id", e.TargetID),
		log.String("target.old_value", e.OldValue),
		log.String("target.new_value", e.NewValue),
		log.String("source.ip", e.SourceIP),
		log.Int("source.port", e.SourcePort),
		log.String("destination.ip", e.DestIP),
		log.Int("destination.port", e.DestPort),
		log.String("source.host", e.SourceHost),
		log.String("destination.host", e.DestHost),
		log.String("user_agent", e.UserAgent),
		log.String("request.id", e.RequestID),
		log.String("service.name", e.Service),
		log.String("service.component", e.Component),
		log.String("deployment.environment", e.Environment),
		log.String("service.version", e.Version),
		log.String("audit.schema_version", e.SchemaVersion),
		log.String("audit.correlation_id", e.CorrelationID),
	}
	if e.TraceID != "" {
		attrs = append(attrs, log.String("trace_id", e.TraceID))
	}
	if e.SpanID != "" {
		attrs = append(attrs, log.String("span_id", e.SpanID))
	}
	for k, v := range e.Attributes {
		attrs = append(attrs, log.String("audit.attr."+k, v))
	}
	return attrs
}

func severityToOTelSeverity(severity Severity) log.Severity {
	switch severity {
	case SeverityVeryHigh:
		return log.SeverityFatal3
	case SeverityHigh:
		return log.SeverityError
	case SeverityMedium:
		return log.SeverityWarn
	case SeverityLow:
		return log.SeverityInfo
	default:
		return log.SeverityUndefined
	}
}
