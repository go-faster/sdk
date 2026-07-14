package audit_test

import "github.com/go-faster/sdk/audit"

func fullEvent() audit.Event {
	e := minimalEvent()
	e.CorrelationID = "req-abc123"
	e.TraceID = "00112233445566778899aabbccddeeff"
	e.SpanID = "0011223344556677"
	e.SessionID = "session-1"
	e.AuthMethod = "password"
	e.Reason = "allowed"
	e.TargetType = "user"
	e.TargetID = "bob"
	e.OldValue = "disabled"
	e.NewValue = "enabled"
	e.SourceIP = "192.0.2.5"
	e.SourcePort = 443
	e.DestIP = "10.0.0.1"
	e.DestPort = 3389
	e.SourceHost = "client"
	e.DestHost = "server"
	e.UserAgent = "test-agent"
	e.RequestID = "req-1"
	e.Attributes = map[string]string{"cat": "Authentication", "proto": "HTTPS"}
	return e
}

func escapingEvent() audit.Event {
	e := fullEvent()
	e.Action = `log\in`
	e.Message = `User | login`
	e.Reason = `bad "thing" \ close ]`
	e.Attributes = map[string]string{"expr": `a=b|c\d`}
	return e
}

func trailingSpaceEvent() audit.Event {
	e := fullEvent()
	e.Message = "User login "
	e.Attributes = map[string]string{"note": "value "}
	return e
}

func refExampleEvent() audit.Event {
	e := minimalEvent()
	e.ID = "200"
	e.Message = "User login"
	e.SourceIP = "192.0.2.5"
	e.SourcePort = 443
	e.Result = audit.ResultSuccess
	e.ActorID = "alice"
	e.TargetType = "service"
	e.TargetID = "WebServer"
	e.Attributes = map[string]string{"cat": "Authentication", "proto": "HTTPS"}
	return e
}
