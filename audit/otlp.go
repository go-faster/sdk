package audit

import (
	"context"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
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
	r.SetBody(attribute.StringValue(e.Message))
	r.AddAttributes(eventToLogAttributes(e)...)
	return r
}

func eventToLogAttributes(e Event) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("event.id", e.ID),
		attribute.String("event.type", string(e.Type)),
		attribute.String("event.action", e.Action),
		attribute.String("event.result", string(e.Result)),
		attribute.Int("event.severity", int(e.Severity)),
		attribute.String("event.reason", e.Reason),
		attribute.String("actor.id", e.ActorID),
		attribute.String("actor.type", string(e.ActorType)),
		attribute.String("actor.session_id", e.SessionID),
		attribute.String("actor.auth_method", e.AuthMethod),
		attribute.String("target.type", e.TargetType),
		attribute.String("target.id", e.TargetID),
		attribute.String("target.old_value", e.OldValue),
		attribute.String("target.new_value", e.NewValue),
		attribute.String("source.ip", e.SourceIP),
		attribute.Int("source.port", e.SourcePort),
		attribute.String("destination.ip", e.DestIP),
		attribute.Int("destination.port", e.DestPort),
		attribute.String("source.host", e.SourceHost),
		attribute.String("destination.host", e.DestHost),
		attribute.String("user_agent", e.UserAgent),
		attribute.String("request.id", e.RequestID),
		attribute.String("service.name", e.Service),
		attribute.String("service.component", e.Component),
		attribute.String("deployment.environment", e.Environment),
		attribute.String("service.version", e.Version),
		attribute.String("audit.schema_version", e.SchemaVersion),
		attribute.String("audit.correlation_id", e.CorrelationID),
	}
	if e.TraceID != "" {
		attrs = append(attrs, attribute.String("trace_id", e.TraceID))
	}
	if e.SpanID != "" {
		attrs = append(attrs, attribute.String("span_id", e.SpanID))
	}
	for k, v := range e.Attributes {
		attrs = append(attrs, attribute.String("audit.attr."+k, v))
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
