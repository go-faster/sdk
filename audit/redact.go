package audit

import (
	"crypto/sha256"
	"encoding/hex"
)

// Redactor redacts an Event in place.
type Redactor interface{ Redact(e *Event) }

type redactorFunc func(e *Event)

func (f redactorFunc) Redact(e *Event) { f(e) }

// NoRedact returns a no-op redactor.
func NoRedact() Redactor { return redactorFunc(func(e *Event) {}) }

// HashFields hashes top-level Event string fields by Go field name and matching attribute keys.
func HashFields(names ...string) Redactor {
	return fieldRedactor(names, func(v string) string {
		sum := sha256.Sum256([]byte(v))
		return hex.EncodeToString(sum[:])
	})
}

// MaskFields masks top-level Event string fields by Go field name and matching attribute keys.
func MaskFields(names ...string) Redactor {
	return fieldRedactor(names, func(string) string { return "***" })
}

// TruncateFields truncates top-level Event string fields by Go field name and matching attribute keys.
func TruncateFields(n int, names ...string) Redactor {
	if n < 0 {
		n = 0
	}
	return fieldRedactor(names, func(v string) string {
		if len(v) <= n {
			return v
		}
		return v[:n]
	})
}

// ChainRedactors chains redactors.
func ChainRedactors(rs ...Redactor) Redactor {
	return redactorFunc(func(e *Event) {
		for _, r := range rs {
			if r != nil {
				r.Redact(e)
			}
		}
	})
}

func fieldRedactor(names []string, fn func(string) string) Redactor {
	return redactorFunc(func(e *Event) {
		if e == nil {
			return
		}
		for _, name := range names {
			redactField(e, name, fn)
			if e.Attributes != nil {
				if v, ok := e.Attributes[name]; ok {
					e.Attributes[name] = fn(v)
				}
			}
		}
	})
}

func redactField(e *Event, name string, fn func(string) string) {
	switch name {
	case "ID":
		e.ID = fn(e.ID)
	case "SchemaVersion":
		e.SchemaVersion = fn(e.SchemaVersion)
	case "Action":
		e.Action = fn(e.Action)
	case "CorrelationID":
		e.CorrelationID = fn(e.CorrelationID)
	case "TraceID":
		e.TraceID = fn(e.TraceID)
	case "SpanID":
		e.SpanID = fn(e.SpanID)
	case "ActorID":
		e.ActorID = fn(e.ActorID)
	case "SessionID":
		e.SessionID = fn(e.SessionID)
	case "AuthMethod":
		e.AuthMethod = fn(e.AuthMethod)
	case "Reason":
		e.Reason = fn(e.Reason)
	case "Message":
		e.Message = fn(e.Message)
	case "TargetType":
		e.TargetType = fn(e.TargetType)
	case "TargetID":
		e.TargetID = fn(e.TargetID)
	case "OldValue":
		e.OldValue = fn(e.OldValue)
	case "NewValue":
		e.NewValue = fn(e.NewValue)
	case "SourceIP":
		e.SourceIP = fn(e.SourceIP)
	case "DestIP":
		e.DestIP = fn(e.DestIP)
	case "SourceHost":
		e.SourceHost = fn(e.SourceHost)
	case "DestHost":
		e.DestHost = fn(e.DestHost)
	case "UserAgent":
		e.UserAgent = fn(e.UserAgent)
	case "RequestID":
		e.RequestID = fn(e.RequestID)
	case "Service":
		e.Service = fn(e.Service)
	case "Component":
		e.Component = fn(e.Component)
	case "Environment":
		e.Environment = fn(e.Environment)
	case "Version":
		e.Version = fn(e.Version)
	}
}
