package audit

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SyslogEncodeConfig configures RFC 5424 encoding.
type SyslogEncodeConfig struct {
	Facility Facility
	Hostname string
	AppName  string
	ProcID   string
	SDID     string
	Clock    func() time.Time
}

// EncodeRFC5424 encodes Event as RFC 5424 syslog without transport framing.
func EncodeRFC5424(e Event, cfg SyslogEncodeConfig) ([]byte, error) {
	if cfg.Facility == 0 {
		cfg.Facility = DefaultSyslogFacility
	}
	if cfg.SDID == "" {
		cfg.SDID = DefaultSyslogSDID
	}
	t := e.Time
	if t.IsZero() {
		if cfg.Clock != nil {
			t = cfg.Clock()
		} else {
			t = time.Now()
		}
	}
	pri := int(cfg.Facility)*8 + auditSeverityToSyslogSeverity(e.Severity, e.Result)
	var b bytes.Buffer
	fmt.Fprintf(&b, "<%d>1 %s %s %s %s audit ", pri, t.UTC().Format("2006-01-02T15:04:05.000000Z07:00"), nilValue(cfg.Hostname), nilValue(cfg.AppName), nilValue(cfg.ProcID))
	b.WriteString(structuredData(e, cfg.SDID))
	if e.Message != "" {
		b.WriteByte(' ')
		b.WriteString(e.Message)
	}
	return b.Bytes(), nil
}

func structuredData(e Event, sdid string) string {
	params := []sdParam{
		{"id", e.ID}, {"type", string(e.Type)}, {"action", e.Action}, {"result", string(e.Result)}, {"actor", e.ActorID},
		{"actor_type", string(e.ActorType)}, {"service", e.Service}, {"component", e.Component}, {"env", e.Environment}, {"schema", e.SchemaVersion},
	}
	optional := []sdParam{
		{"act", e.Action}, {"reason", e.Reason}, {"correlation_id", e.CorrelationID}, {"target_type", e.TargetType}, {"target", e.TargetID}, {"old_value", e.OldValue},
		{"new_value", e.NewValue}, {"src", e.SourceIP}, {"dst", e.DestIP}, {"shost", e.SourceHost}, {"dhost", e.DestHost},
		{"user_agent", e.UserAgent}, {"request_id", e.RequestID}, {"session_id", e.SessionID}, {"auth_method", e.AuthMethod},
	}
	if e.SourcePort != 0 {
		optional = append(optional, sdParam{"src_port", strconv.Itoa(e.SourcePort)})
	}
	if e.DestPort != 0 {
		optional = append(optional, sdParam{"dst_port", strconv.Itoa(e.DestPort)})
	}
	optional = append(optional, sdParam{"trace_id", e.TraceID})
	optional = append(optional, sdParam{"span_id", e.SpanID})
	params = append(params, optionalNonEmpty(optional)...)
	blocks := []string{sdBlock(sdid, params)}
	if len(e.Attributes) > 0 {
		var ext []sdParam
		keys := sortedKeys(e.Attributes)
		for _, k := range keys {
			ext = append(ext, sdParam{k, e.Attributes[k]})
		}
		blocks = append(blocks, sdBlock(sdid+"-ext", ext))
	}
	if len(blocks) == 0 {
		return "-"
	}
	return strings.Join(blocks, "")
}

type sdParam struct{ key, value string }

func sdBlock(sdid string, params []sdParam) string {
	var b strings.Builder
	b.WriteByte('[')
	b.WriteString(sdid)
	for _, p := range params {
		b.WriteByte(' ')
		b.WriteString(p.key)
		b.WriteString("=\"")
		b.WriteString(escapeSDValue(p.value))
		b.WriteByte('"')
	}
	b.WriteByte(']')
	return b.String()
}

func optionalNonEmpty(in []sdParam) []sdParam {
	out := in[:0]
	for _, p := range in {
		if p.value != "" {
			out = append(out, p)
		}
	}
	return out
}

func escapeSDValue(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, `]`, `\]`)
	return s
}

func nilValue(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func auditSeverityToSyslogSeverity(severity Severity, result Result) int {
	switch severity {
	case SeverityVeryHigh:
		if result == ResultFailure {
			return 1
		}
		return 0
	case SeverityHigh:
		if result == ResultFailure {
			return 3
		}
		return 2
	case SeverityMedium:
		if result == ResultFailure {
			return 4
		}
		return 5
	default:
		return 6
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
