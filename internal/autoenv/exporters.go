// Package autoenv provides helpers for parsing OpenTelemetry environment variables.
package autoenv

import (
	"slices"
	"strings"

	"github.com/go-faster/errors"
)

// ExporterNone is a name of the no-op exporter.
const ExporterNone = "none"

// ParseExporters parses comma-separated exporter list, like OTEL_TRACES_EXPORTER.
//
// Empty entries are ignored, duplicates are removed. [ExporterNone] is only
// valid as the only entry.
func ParseExporters(v string) ([]string, error) {
	var names []string
	for name := range strings.SplitSeq(v, ",") {
		name = strings.TrimSpace(name)
		if name == "" || slices.Contains(names, name) {
			continue
		}
		names = append(names, name)
	}
	switch {
	case len(names) == 0:
		return nil, errors.New("no exporters")
	case len(names) > 1 && slices.Contains(names, ExporterNone):
		return nil, errors.Errorf("%q exporter can't be used with other exporters", ExporterNone)
	}
	return names, nil
}
