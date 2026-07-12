package audit

import (
	"context"
	"io"
	"sync"

	"github.com/go-faster/errors"
	"go.opentelemetry.io/otel/sdk/resource"
)

// CEFTransport transports encoded CEF records.
type CEFTransport interface {
	writeCEF(ctx context.Context, raw []byte) error
	closeCEF(ctx context.Context) error
}

type cefWriterTransport struct {
	out io.Writer
	mu  sync.Mutex
}

func (c *cefWriterTransport) writeCEF(ctx context.Context, raw []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.out.Write(append(append([]byte{}, raw...), '\n'))
	return err
}

func (c *cefWriterTransport) closeCEF(ctx context.Context) error { return nil }

type cefSyslogTransport struct{ syslog *SyslogExporter }

func (c cefSyslogTransport) writeCEF(ctx context.Context, raw []byte) error {
	if c.syslog == nil {
		return errors.New("audit: syslog exporter is nil")
	}
	return c.syslog.syslogWrite(ctx, raw)
}
func (c cefSyslogTransport) closeCEF(ctx context.Context) error { return nil }

// CEFOverSyslog sends raw CEF over a SyslogExporter transport.
func CEFOverSyslog(s *SyslogExporter) CEFTransport { return cefSyslogTransport{syslog: s} }

// CEFOverWriter writes raw CEF lines to a writer.
func CEFOverWriter(w io.Writer) CEFTransport { return &cefWriterTransport{out: w} }

type cefExporterConfig struct {
	vendor         string
	product        string
	version        string
	resource       *resource.Resource
	transport      CEFTransport
	facility       Facility
	severityMapper func(Severity) (string, error)
	maxLen         int
}

// CEFExporterOption configures CEFExporter.
type CEFExporterOption interface {
	applyCEF(cefExporterConfig) cefExporterConfig
}

type cefExporterOptionFunc func(cefExporterConfig) cefExporterConfig

func (f cefExporterOptionFunc) applyCEF(conf cefExporterConfig) cefExporterConfig { return f(conf) }

// CEFExporter exports CEF records.
type CEFExporter struct{ cfg cefExporterConfig }

// NewCEFExporter creates a CEFExporter.
//
// Device fields default from the [resource.Resource] (vendor <- device.manufacturer,
// product <- service.name, version <- service.version) when set via
// [WithCEFResource]; [WithCEFDevice] takes precedence over resource defaults.
func NewCEFExporter(opts ...CEFExporterOption) (*CEFExporter, error) {
	cfg := cefExporterConfig{maxLen: 8192, facility: DefaultSyslogFacility}
	for _, o := range opts {
		cfg = o.applyCEF(cfg)
	}
	if cfg.resource != nil {
		rs := fromResource(cfg.resource)
		if cfg.vendor == "" {
			cfg.vendor = rs.vendor
		}
		if cfg.product == "" {
			cfg.product = rs.service
		}
		if cfg.version == "" {
			cfg.version = rs.version
		}
	}
	if cfg.vendor == "" || cfg.product == "" || cfg.version == "" {
		return nil, errors.New("audit: CEF device is required")
	}
	if cfg.transport == nil {
		return nil, errors.New("audit: CEF transport is required (use CEFOverSyslog or CEFOverWriter)")
	}
	return &CEFExporter{cfg: cfg}, nil
}

// WithCEFDevice sets CEF device fields. Takes precedence over [WithCEFResource].
func WithCEFDevice(vendor, product, version string) CEFExporterOption {
	return cefExporterOptionFunc(func(conf cefExporterConfig) cefExporterConfig {
		conf.vendor, conf.product, conf.version = vendor, product, version
		return conf
	})
}

// WithCEFResource sets the OpenTelemetry [resource.Resource] used to populate
// CEF device defaults: vendor <- device.manufacturer, product <- service.name,
// version <- service.version. Explicit [WithCEFDevice] wins over resource.
func WithCEFResource(r *resource.Resource) CEFExporterOption {
	return cefExporterOptionFunc(func(conf cefExporterConfig) cefExporterConfig {
		conf.resource = r
		return conf
	})
}

// WithCEFTransport sets CEF transport.
func WithCEFTransport(t CEFTransport) CEFExporterOption {
	return cefExporterOptionFunc(func(conf cefExporterConfig) cefExporterConfig { conf.transport = t; return conf })
}

// WithCEFFacility sets CEF syslog facility.
func WithCEFFacility(f Facility) CEFExporterOption {
	return cefExporterOptionFunc(func(conf cefExporterConfig) cefExporterConfig { conf.facility = f; return conf })
}

// WithCEFSeverityMapper sets severity mapper.
func WithCEFSeverityMapper(fn func(Severity) (string, error)) CEFExporterOption {
	return cefExporterOptionFunc(func(conf cefExporterConfig) cefExporterConfig { conf.severityMapper = fn; return conf })
}

// WithCEFMaxLen sets max encoded length.
func WithCEFMaxLen(n int) CEFExporterOption {
	return cefExporterOptionFunc(func(conf cefExporterConfig) cefExporterConfig { conf.maxLen = n; return conf })
}

// DeviceVendor returns the resolved CEF device vendor (from [WithCEFDevice]
// or [WithCEFResource]).
func (c *CEFExporter) DeviceVendor() string { return c.cfg.vendor }

// DeviceProduct returns the resolved CEF device product.
func (c *CEFExporter) DeviceProduct() string { return c.cfg.product }

// DeviceVersion returns the resolved CEF device version.
func (c *CEFExporter) DeviceVersion() string { return c.cfg.version }

// Export implements Exporter.
func (c *CEFExporter) Export(ctx context.Context, events []Event) error {
	for _, event := range events {
		out, err := EncodeCEF(event, CEFEncodeConfig{
			DeviceVendor:   c.cfg.vendor,
			DeviceProduct:  c.cfg.product,
			DeviceVersion:  c.cfg.version,
			SeverityMapper: c.cfg.severityMapper,
			MaxLen:         c.cfg.maxLen,
		})
		if err != nil {
			return err
		}
		if err := c.cfg.transport.writeCEF(ctx, out); err != nil {
			return errors.Wrap(err, "audit: write cef")
		}
	}
	return nil
}

// Close implements Exporter.
func (c *CEFExporter) Close(ctx context.Context) error { return c.cfg.transport.closeCEF(ctx) }

// Name implements Exporter.
func (c *CEFExporter) Name() string { return "cef" }
