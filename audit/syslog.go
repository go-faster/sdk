package audit

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/go-faster/errors"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap"

	"github.com/go-faster/sdk/zctx"
)

// SyslogTransport is syslog network transport.
type SyslogTransport int

// Syslog transports.
const (
	SyslogTCP SyslogTransport = iota
	SyslogTCP_TLS
	SyslogUDP
)

// Facility is a syslog facility.
type Facility uint8

// Syslog facilities.
const (
	LOG_USER     Facility = 1
	LOG_AUTH     Facility = 4
	LOG_AUTHPRIV Facility = 10
	LOG_DAEMON   Facility = 3
	LOG_LOCAL0   Facility = 16
	LOG_LOCAL1   Facility = 17
	LOG_LOCAL2   Facility = 18
	LOG_LOCAL3   Facility = 19
	LOG_LOCAL4   Facility = 20
	LOG_LOCAL5   Facility = 21
	LOG_LOCAL6   Facility = 22
	LOG_LOCAL7   Facility = 23
)

// Defaults.
const (
	DefaultSyslogFacility Facility = LOG_AUTH
	DefaultSyslogSDID              = "audit@8734"
)

type syslogExporterConfig struct {
	transport      SyslogTransport
	tlsConfig      *tls.Config
	facility       Facility
	hostname       string
	appName        string
	procID         string
	sdid           string
	resource       *resource.Resource
	connectTimeout time.Duration
	reconnect      bool
}

// SyslogExporterOption configures SyslogExporter.
type SyslogExporterOption interface {
	applySyslog(syslogExporterConfig) syslogExporterConfig
}

type syslogExporterOptionFunc func(syslogExporterConfig) syslogExporterConfig

func (f syslogExporterOptionFunc) applySyslog(conf syslogExporterConfig) syslogExporterConfig {
	return f(conf)
}

// SyslogExporter exports RFC 5424 syslog. TLS uses octet-counting framing; plain TCP uses newline framing.
type SyslogExporter struct {
	addr string
	cfg  syslogExporterConfig
	mu   sync.Mutex
	conn net.Conn
}

// NewSyslogExporter creates a SyslogExporter.
//
// Hostname defaults from host.name and AppName from service.name when
// [WithSyslogResource] is used; [WithSyslogHostname] and [WithSyslogAppName]
// take precedence over resource defaults.
func NewSyslogExporter(addr string, opts ...SyslogExporterOption) (*SyslogExporter, error) {
	cfg := syslogExporterConfig{facility: DefaultSyslogFacility, sdid: DefaultSyslogSDID, connectTimeout: 5 * time.Second, reconnect: true}
	for _, o := range opts {
		cfg = o.applySyslog(cfg)
	}
	if cfg.resource != nil {
		rs := fromResource(cfg.resource)
		if cfg.hostname == "" {
			cfg.hostname = rs.host
		}
		if cfg.appName == "" {
			cfg.appName = rs.service
		}
	}
	return &SyslogExporter{addr: addr, cfg: cfg}, nil
}

// WithSyslogTransport sets syslog transport.
func WithSyslogTransport(t SyslogTransport) SyslogExporterOption {
	return syslogExporterOptionFunc(func(conf syslogExporterConfig) syslogExporterConfig { conf.transport = t; return conf })
}

// WithSyslogTLSConfig sets TLS config.
func WithSyslogTLSConfig(c *tls.Config) SyslogExporterOption {
	return syslogExporterOptionFunc(func(conf syslogExporterConfig) syslogExporterConfig { conf.tlsConfig = c; return conf })
}

// WithSyslogFacility sets syslog facility.
func WithSyslogFacility(f Facility) SyslogExporterOption {
	return syslogExporterOptionFunc(func(conf syslogExporterConfig) syslogExporterConfig { conf.facility = f; return conf })
}

// WithSyslogHostname sets hostname.
func WithSyslogHostname(h string) SyslogExporterOption {
	return syslogExporterOptionFunc(func(conf syslogExporterConfig) syslogExporterConfig { conf.hostname = h; return conf })
}

// WithSyslogAppName sets APP-NAME.
func WithSyslogAppName(s string) SyslogExporterOption {
	return syslogExporterOptionFunc(func(conf syslogExporterConfig) syslogExporterConfig { conf.appName = s; return conf })
}

// WithSyslogProcID sets PROCID.
func WithSyslogProcID(s string) SyslogExporterOption {
	return syslogExporterOptionFunc(func(conf syslogExporterConfig) syslogExporterConfig { conf.procID = s; return conf })
}

// WithSyslogSDID sets structured data ID.
func WithSyslogSDID(id string) SyslogExporterOption {
	return syslogExporterOptionFunc(func(conf syslogExporterConfig) syslogExporterConfig { conf.sdid = id; return conf })
}

// WithSyslogResource sets the OpenTelemetry [resource.Resource] used to populate
// Hostname (host.name) and AppName (service.name) defaults. Explicit
// [WithSyslogHostname] and [WithSyslogAppName] take precedence.
func WithSyslogResource(r *resource.Resource) SyslogExporterOption {
	return syslogExporterOptionFunc(func(conf syslogExporterConfig) syslogExporterConfig {
		conf.resource = r
		return conf
	})
}

// WithSyslogConnectTimeout sets connect timeout.
func WithSyslogConnectTimeout(d time.Duration) SyslogExporterOption {
	return syslogExporterOptionFunc(func(conf syslogExporterConfig) syslogExporterConfig { conf.connectTimeout = d; return conf })
}

// WithSyslogReconnect sets reconnect behavior.
func WithSyslogReconnect(b bool) SyslogExporterOption {
	return syslogExporterOptionFunc(func(conf syslogExporterConfig) syslogExporterConfig { conf.reconnect = b; return conf })
}

// Hostname returns the resolved syslog HOSTNAME (from [WithSyslogHostname]
// or [WithSyslogResource]).
func (s *SyslogExporter) Hostname() string { return s.cfg.hostname }

// AppName returns the resolved syslog APP-NAME.
func (s *SyslogExporter) AppName() string { return s.cfg.appName }

// Export implements Exporter.
func (s *SyslogExporter) Export(ctx context.Context, events []Event) error {
	for _, e := range events {
		msg, err := EncodeRFC5424(e, SyslogEncodeConfig{Facility: s.cfg.facility, Hostname: s.cfg.hostname, AppName: s.cfg.appName, ProcID: s.cfg.procID, SDID: s.cfg.sdid})
		if err != nil {
			return err
		}
		if s.cfg.transport == SyslogUDP && len(msg) > 8192 {
			zctx.From(ctx).Warn("audit: udp message too large, dropping", zap.Int("size", len(msg)), zap.String("event_id", e.ID))
			continue
		}
		if err := s.syslogWrite(ctx, msg); err != nil {
			return errors.Wrapf(err, "audit: write %s", s.addr)
		}
	}
	return nil
}

func (s *SyslogExporter) syslogWrite(ctx context.Context, raw []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureConn(); err != nil {
		return err
	}
	data := raw
	if s.cfg.transport == SyslogTCP_TLS {
		data = []byte(fmt.Sprintf("%d %s", len(raw), raw))
	} else if s.cfg.transport == SyslogTCP {
		data = append(append([]byte{}, raw...), '\n')
	}
	_, err := s.conn.Write(data)
	if err == nil || !s.cfg.reconnect || s.cfg.transport == SyslogUDP {
		return err
	}
	_ = s.conn.Close()
	s.conn = nil
	if err := s.ensureConn(); err != nil {
		return err
	}
	_, err = s.conn.Write(data)
	return err
}

func (s *SyslogExporter) ensureConn() error {
	if s.conn != nil {
		return nil
	}
	dialer := net.Dialer{Timeout: s.cfg.connectTimeout}
	if s.cfg.transport == SyslogTCP_TLS {
		conn, err := tls.DialWithDialer(&dialer, "tcp", s.addr, s.cfg.tlsConfig)
		s.conn = conn
		return err
	}
	network := "tcp"
	if s.cfg.transport == SyslogUDP {
		network = "udp"
	}
	conn, err := dialer.Dial(network, s.addr)
	s.conn = conn
	return err
}

// Close implements Exporter.
func (s *SyslogExporter) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return nil
	}
	if err := s.conn.Close(); err != nil {
		return errors.Wrap(err, "audit: close syslog")
	}
	s.conn = nil
	return nil
}

// Name implements Exporter.
func (s *SyslogExporter) Name() string { return "syslog" }
