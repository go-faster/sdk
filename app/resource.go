package app

import (
	"context"
	"slices"

	"github.com/go-faster/errors"
	"go.opentelemetry.io/otel/sdk/resource"
)

func defaultResourceOptions() []resource.Option {
	return []resource.Option{
		resource.WithProcessRuntimeDescription(),
		resource.WithProcessRuntimeVersion(),
		resource.WithProcessRuntimeName(),
		resource.WithOS(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithProcessPID(),
		resource.WithProcessExecutableName(),
		resource.WithProcessExecutablePath(),
		resource.WithProcessCommandArgs(),
	}
}

// newResource builds resource from given options.
//
// [resource.WithFromEnv] is always applied last, so OTEL_SERVICE_NAME and
// OTEL_RESOURCE_ATTRIBUTES take precedence over attributes set programmatically.
func newResource(ctx context.Context, opts []resource.Option) (*resource.Resource, error) {
	opts = append(slices.Clip(opts), resource.WithFromEnv())
	r, err := resource.New(ctx, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "new")
	}
	return resource.Merge(resource.Default(), r)
}

// Resource returns new resource for application.
//
// Combines following detectors:
// - ProcessRuntimeDescription
// - ProcessRuntimeVersion
// - ProcessRuntimeName
// And merges it with default resource.
//
// Deprecated: use [WithResourceOptions], [WithServiceName], [WithServiceNamespace].
func Resource(ctx context.Context) (*resource.Resource, error) {
	opts := []resource.Option{
		resource.WithProcessRuntimeDescription(),
		resource.WithProcessRuntimeVersion(),
		resource.WithProcessRuntimeName(),
	}
	r, err := resource.New(ctx, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "new")
	}
	return resource.Merge(resource.Default(), r)
}
