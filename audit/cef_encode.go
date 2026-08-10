package audit

import (
	"bytes"
	"strconv"
	"strings"
)

// CEFEncodeConfig configures CEF encoding.
type CEFEncodeConfig struct {
	DeviceVendor   string
	DeviceProduct  string
	DeviceVersion  string
	SeverityMapper func(Severity) (string, error)
	MaxLen         int
}

// EncodeCEF encodes Event as CEF. SignatureID uses Event.ID; extra attributes use available custom fields, then _audit_attr_ keys.
func EncodeCEF(e Event, cfg CEFEncodeConfig) ([]byte, error) {
	severityMapper := cfg.SeverityMapper
	if severityMapper == nil {
		severityMapper = defaultCEFSeverity
	}
	sev, err := severityMapper(e.Severity)
	if err != nil {
		return nil, err
	}
	name := e.Message
	if name == "" {
		name = string(e.Type)
	}
	var b bytes.Buffer
	b.WriteString("CEF:1|")
	b.WriteString(escapeCEFHeader(cfg.DeviceVendor))
	b.WriteByte('|')
	b.WriteString(escapeCEFHeader(cfg.DeviceProduct))
	b.WriteByte('|')
	b.WriteString(escapeCEFHeader(cfg.DeviceVersion))
	b.WriteByte('|')
	b.WriteString(escapeCEFHeader(e.ID))
	b.WriteByte('|')
	b.WriteString(escapeCEFHeader(name))
	b.WriteByte('|')
	b.WriteString(escapeCEFHeader(sev))
	b.WriteByte('|')
	b.WriteString(cefExtension(e))
	out := b.Bytes()
	if cfg.MaxLen > 0 && len(out) > cfg.MaxLen {
		out = out[:cfg.MaxLen]
	}
	return out, nil
}

func cefExtension(e Event) string {
	fields := make([]kv, 0, 32)
	add := func(k, v string) {
		if v != "" {
			fields = append(fields, kv{k, v})
		}
	}
	add("act", e.Action)
	add("app", e.Service)
	custom := newCEFCustomSlots()
	custom.add("Correlation ID", e.CorrelationID)
	if e.TargetID != "" {
		switch strings.ToLower(e.TargetType) {
		case "user", "account", "principal":
			add("duser", e.TargetID)
		case "file", "path":
			add("file", e.TargetID)
		default:
			custom.add("Target", e.TargetID)
		}
	}
	custom.add("Auth Method", e.AuthMethod)
	custom.add("Service", e.Service)
	custom.add("Component", e.Component)
	custom.add("Session ID", e.SessionID)
	custom.add("Environment", e.Environment)
	fields = append(fields, custom.fields...)
	add("dst", e.DestIP)
	add("dhost", e.DestHost)
	if e.DestPort != 0 {
		add("dpt", strconv.Itoa(e.DestPort))
	}
	add("externalId", e.ID)
	add("msg", e.Message)
	add("outcome", string(e.Result))
	add("reason", e.Reason)
	if !e.Time.IsZero() {
		add("rt", strconv.FormatInt(e.Time.UnixMilli(), 10))
	}
	add("src", e.SourceIP)
	add("shost", e.SourceHost)
	if e.SourcePort != 0 {
		add("spt", strconv.Itoa(e.SourcePort))
	}
	add("suser", e.ActorID)
	add("trace_id", e.TraceID)
	add("span_id", e.SpanID)
	for _, k := range sortedKeys(e.Attributes) {
		if custom.next <= 6 {
			fields = append(fields, kv{key: "cs" + strconv.Itoa(custom.next) + "Label", value: k})
			fields = append(fields, kv{key: "cs" + strconv.Itoa(custom.next), value: e.Attributes[k]})
			custom.next++
			continue
		}
		fields = append(fields, kv{key: "_audit_attr_" + k, value: e.Attributes[k]})
	}
	return joinCEFFields(fields)
}

type kv struct{ key, value string }

type cefCustomSlots struct {
	next   int
	fields []kv
}

func newCEFCustomSlots() cefCustomSlots { return cefCustomSlots{next: 1} }

func (c *cefCustomSlots) add(label, value string) {
	if value == "" || c.next > 6 {
		return
	}
	n := strconv.Itoa(c.next)
	c.fields = append(c.fields, kv{"cs" + n + "Label", label}, kv{"cs" + n, value})
	c.next++
}

func joinCEFFields(fields []kv) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		value := strings.TrimRight(field.value, " ")
		if value == "" {
			continue
		}
		parts = append(parts, field.key+"="+escapeCEFExtension(value))
	}
	return strings.Join(parts, " ")
}

func escapeCEFHeader(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `|`, `\|`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}

func escapeCEFExtension(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `=`, `\=`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}

func defaultCEFSeverity(severity Severity) (string, error) {
	switch severity {
	case SeverityVeryHigh:
		return "10", nil
	case SeverityHigh:
		return "8", nil
	case SeverityMedium:
		return "6", nil
	case SeverityLow:
		return "3", nil
	default:
		return "Unknown", nil
	}
}
