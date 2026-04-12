// Package weaveryaml provides utilities for working with OpenTelemetry Weaver YAML registry/schema files.
package weaveryaml

// Schema represents the structure of a Weaver YAML schema file.
type Schema struct {
	Groups []Group `yaml:"groups"`
}

// Group describes a signal in a Weaver YAML schema file.
type Group struct {
	ID         string `yaml:"id"`
	Type       string `yaml:"type"`
	Instrument string `yaml:"instrument"`
	Brief      string `yaml:"brief,omitempty"`
	MetricName string `yaml:"metric_name,omitempty"`
	Unit       string `yaml:"unit,omitempty"`
}
