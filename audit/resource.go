package audit

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

// resourceStrings extracts provenance defaults from a [resource.Resource] using
// OpenTelemetry semantic-convention attribute keys.
type resourceStrings struct {
	service     string // service.name
	component   string // service.namespace
	version     string // service.version
	environment string // deployment.environment.name
	host        string // host.name
	vendor      string // device.manufacturer
}

func fromResource(r *resource.Resource) resourceStrings {
	var out resourceStrings
	if r == nil {
		return out
	}
	for _, kv := range r.Attributes() {
		switch kv.Key {
		case semconv.ServiceNameKey:
			out.service = kv.Value.AsString()
		case semconv.ServiceNamespaceKey:
			out.component = kv.Value.AsString()
		case semconv.ServiceVersionKey:
			out.version = kv.Value.AsString()
		case semconv.DeploymentEnvironmentNameKey:
			out.environment = kv.Value.AsString()
		case semconv.HostNameKey:
			out.host = kv.Value.AsString()
		case semconv.DeviceManufacturerKey:
			out.vendor = kv.Value.AsString()
		}
	}
	return out
}

// resourceString returns the value for key, or "" if r is nil or key absent.
func resourceString(r *resource.Resource, key attribute.Key) string {
	if r == nil {
		return ""
	}
	for _, kv := range r.Attributes() {
		if kv.Key == key {
			return kv.Value.AsString()
		}
	}
	return ""
}
