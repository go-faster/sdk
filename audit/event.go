package audit

import (
	"context"
	"time"

	"github.com/go-faster/errors"
)

// CurrentSchemaVersion is the current audit event schema version.
const CurrentSchemaVersion = "audit/v1"

// EventType is a normalized audit event type.
type EventType string

// Event types.
const (
	EventCreate  EventType = "create"
	EventRead    EventType = "read"
	EventUpdate  EventType = "update"
	EventDelete  EventType = "delete"
	EventExecute EventType = "execute"
	EventStart   EventType = "start"
	EventStop    EventType = "stop"
	EventEnable  EventType = "enable"
	EventDisable EventType = "disable"
	EventLogin   EventType = "login"
	EventLogout  EventType = "logout"
)

// Result is the outcome of an audit event.
type Result string

// Event results.
const (
	ResultSuccess Result = "success"
	ResultFailure Result = "failure"
	ResultPartial Result = "partial"
)

// Severity is a normalized audit event severity.
type Severity uint8

// Event severities.
const (
	SeverityUnknown  Severity = 0
	SeverityLow      Severity = 3
	SeverityMedium   Severity = 6
	SeverityHigh     Severity = 8
	SeverityVeryHigh Severity = 10
)

// ActorType is a normalized actor type.
type ActorType string

// Actor types.
const (
	ActorUser    ActorType = "user"
	ActorService ActorType = "service"
	ActorSystem  ActorType = "system"
	ActorAdmin   ActorType = "admin"
)

// Event is a normalized audit event.
type Event struct {
	ID, SchemaVersion string
	Type              EventType
	Action            string
	CorrelationID     string
	TraceID           string
	SpanID            string
	Time              time.Time
	ActorID           string
	ActorType         ActorType
	SessionID         string
	AuthMethod        string
	Result            Result
	Reason            string
	Severity          Severity
	Message           string
	TargetType        string
	TargetID          string
	OldValue          string
	NewValue          string
	SourceIP          string
	SourcePort        int
	DestIP            string
	DestPort          int
	SourceHost        string
	DestHost          string
	UserAgent         string
	RequestID         string
	Service           string
	Component         string
	Environment       string
	Version           string
	Attributes        map[string]string
}

// EventBuilder builds an Event.
type EventBuilder struct{ e Event }

// NewEvent creates a new EventBuilder.
func NewEvent(t EventType, actorID string, result Result) *EventBuilder {
	return &EventBuilder{e: Event{Type: t, ActorID: actorID, Result: result}}
}

// ID sets event ID.
func (b *EventBuilder) ID(v string) *EventBuilder { b.e.ID = v; return b }

// SchemaVersion sets schema version.
func (b *EventBuilder) SchemaVersion(v string) *EventBuilder { b.e.SchemaVersion = v; return b }

// Type sets event type.
func (b *EventBuilder) Type(v EventType) *EventBuilder { b.e.Type = v; return b }

// Action sets event action.
func (b *EventBuilder) Action(v string) *EventBuilder { b.e.Action = v; return b }

// CorrelationID sets correlation ID.
func (b *EventBuilder) CorrelationID(v string) *EventBuilder { b.e.CorrelationID = v; return b }

// TraceID sets the trace id. Only populate this from a real OpenTelemetry span
// context (see [WithTraceEnrichment]); do not synthesize it from other IDs.
func (b *EventBuilder) TraceID(v string) *EventBuilder { b.e.TraceID = v; return b }

// SpanID sets the span id. Only populate this from a real OpenTelemetry span
// context (see [WithTraceEnrichment]); do not synthesize it from other IDs.
func (b *EventBuilder) SpanID(v string) *EventBuilder { b.e.SpanID = v; return b }

// Time sets event time.
func (b *EventBuilder) Time(v time.Time) *EventBuilder { b.e.Time = v; return b }

// ActorID sets actor ID.
func (b *EventBuilder) ActorID(v string) *EventBuilder { b.e.ActorID = v; return b }

// ActorType sets actor type.
func (b *EventBuilder) ActorType(v ActorType) *EventBuilder { b.e.ActorType = v; return b }

// SessionID sets session ID.
func (b *EventBuilder) SessionID(v string) *EventBuilder { b.e.SessionID = v; return b }

// AuthMethod sets auth method.
func (b *EventBuilder) AuthMethod(v string) *EventBuilder { b.e.AuthMethod = v; return b }

// Result sets event result.
func (b *EventBuilder) Result(v Result) *EventBuilder { b.e.Result = v; return b }

// Reason sets event reason.
func (b *EventBuilder) Reason(v string) *EventBuilder { b.e.Reason = v; return b }

// Severity sets event severity.
func (b *EventBuilder) Severity(v Severity) *EventBuilder { b.e.Severity = v; return b }

// Message sets event message.
func (b *EventBuilder) Message(v string) *EventBuilder { b.e.Message = v; return b }

// TargetType sets target type.
func (b *EventBuilder) TargetType(v string) *EventBuilder { b.e.TargetType = v; return b }

// TargetID sets target ID.
func (b *EventBuilder) TargetID(v string) *EventBuilder { b.e.TargetID = v; return b }

// OldValue sets old value.
func (b *EventBuilder) OldValue(v string) *EventBuilder { b.e.OldValue = v; return b }

// NewValue sets new value.
func (b *EventBuilder) NewValue(v string) *EventBuilder { b.e.NewValue = v; return b }

// SourceIP sets source IP.
func (b *EventBuilder) SourceIP(v string) *EventBuilder { b.e.SourceIP = v; return b }

// SourcePort sets source port.
func (b *EventBuilder) SourcePort(v int) *EventBuilder { b.e.SourcePort = v; return b }

// DestIP sets destination IP.
func (b *EventBuilder) DestIP(v string) *EventBuilder { b.e.DestIP = v; return b }

// DestPort sets destination port.
func (b *EventBuilder) DestPort(v int) *EventBuilder { b.e.DestPort = v; return b }

// SourceHost sets source host.
func (b *EventBuilder) SourceHost(v string) *EventBuilder { b.e.SourceHost = v; return b }

// DestHost sets destination host.
func (b *EventBuilder) DestHost(v string) *EventBuilder { b.e.DestHost = v; return b }

// UserAgent sets user agent.
func (b *EventBuilder) UserAgent(v string) *EventBuilder { b.e.UserAgent = v; return b }

// RequestID sets request ID.
func (b *EventBuilder) RequestID(v string) *EventBuilder { b.e.RequestID = v; return b }

// Service sets service.
func (b *EventBuilder) Service(v string) *EventBuilder { b.e.Service = v; return b }

// Component sets component.
func (b *EventBuilder) Component(v string) *EventBuilder { b.e.Component = v; return b }

// Environment sets environment.
func (b *EventBuilder) Environment(v string) *EventBuilder { b.e.Environment = v; return b }

// Version sets version.
func (b *EventBuilder) Version(v string) *EventBuilder { b.e.Version = v; return b }

// Attributes sets attributes.
func (b *EventBuilder) Attributes(v map[string]string) *EventBuilder { b.e.Attributes = v; return b }

// Attribute sets an attribute.
func (b *EventBuilder) Attribute(k, v string) *EventBuilder {
	if b.e.Attributes == nil {
		b.e.Attributes = map[string]string{}
	}
	b.e.Attributes[k] = v
	return b
}

// Build validates and returns an Event.
func (b *EventBuilder) Build(ctx context.Context) (Event, error) {
	e := b.e
	if e.Type == "" {
		return Event{}, errors.New("audit: event type is required")
	}
	if e.ActorID == "" {
		return Event{}, errors.New("audit: actor id is required")
	}
	if e.Result == "" {
		return Event{}, errors.New("audit: result is required")
	}
	if e.ID == "" {
		e.ID = defaultIDGenerator()
	}
	if e.SchemaVersion == "" {
		e.SchemaVersion = CurrentSchemaVersion
	}
	if e.Time.IsZero() {
		e.Time = defaultClock().UTC()
	} else {
		e.Time = e.Time.UTC()
	}
	return e, nil
}
