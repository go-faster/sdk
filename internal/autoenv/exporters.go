// Package autoenv provides helpers for parsing OpenTelemetry environment variables.
package autoenv

import (
	"os"
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
	names := parseList(v)
	switch {
	case len(names) == 0:
		return nil, errors.New("no exporters")
	case len(names) > 1 && slices.Contains(names, ExporterNone):
		return nil, errors.Errorf("%q exporter can't be used with other exporters", ExporterNone)
	}
	return names, nil
}

// AdditionalEndpoints returns additional OTLP endpoints for the given signal,
// e.g. "TRACES", to fan out telemetry to more than one backend.
//
// Reads comma-separated GOFASTER_OTLP_<SIGNAL>_ENDPOINTS, falling back to
// GOFASTER_OTLP_ENDPOINTS. These are not defined by the OpenTelemetry
// specification, where OTEL_EXPORTER_OTLP_ENDPOINT is singular.
func AdditionalEndpoints(signal string) []string {
	v := os.Getenv("GOFASTER_OTLP_" + signal + "_ENDPOINTS")
	if v == "" {
		v = os.Getenv("GOFASTER_OTLP_ENDPOINTS")
	}
	return parseList(v)
}

func parseList(v string) []string {
	var list []string
	for e := range strings.SplitSeq(v, ",") {
		e = strings.TrimSpace(e)
		if e == "" || slices.Contains(list, e) {
			continue
		}
		list = append(list, e)
	}
	return list
}
