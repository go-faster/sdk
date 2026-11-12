package app

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestDefaultResourceOptionsWithoutUser(t *testing.T) {
	oldUser, ok := os.LookupEnv("USER")
	require.NoError(t, os.Unsetenv("USER"))
	t.Cleanup(func() {
		if ok {
			require.NoError(t, os.Setenv("USER", oldUser))
			return
		}
		require.NoError(t, os.Unsetenv("USER"))
	})

	res, err := resource.New(context.Background(), defaultResourceOptions()...)
	require.NoError(t, err)

	attrs := res.Attributes()
	for _, attr := range attrs {
		require.NotEqual(t, "process.owner", string(attr.Key))
	}
}

func TestNewResourceEnvPrecedence(t *testing.T) {
	newOptions := func(o ...Option) []resource.Option {
		opts := options{resourceOptions: defaultResourceOptions()}
		for _, opt := range o {
			opt.apply(&opts)
		}
		return opts.resourceOptions
	}
	attributes := func(t *testing.T, res *resource.Resource) map[string]string {
		t.Helper()

		m := map[string]string{}
		for _, attr := range res.Attributes() {
			m[string(attr.Key)] = attr.Value.String()
		}
		return m
	}

	t.Run("Programmatic", func(t *testing.T) {
		t.Setenv("OTEL_SERVICE_NAME", "")
		t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")

		res, err := newResource(context.Background(), newOptions(
			WithServiceName("app"),
			WithServiceNamespace("ns"),
		))
		require.NoError(t, err)

		attrs := attributes(t, res)
		require.Equal(t, "app", attrs["service.name"])
		require.Equal(t, "ns", attrs["service.namespace"])
	})
	t.Run("ServiceNameEnv", func(t *testing.T) {
		t.Setenv("OTEL_SERVICE_NAME", "app-from-env")
		t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.namespace=ns-from-env")

		res, err := newResource(context.Background(), newOptions(
			WithServiceName("app"),
			WithServiceNamespace("ns"),
		))
		require.NoError(t, err)

		attrs := attributes(t, res)
		require.Equal(t, "app-from-env", attrs["service.name"])
		require.Equal(t, "ns-from-env", attrs["service.namespace"])
	})
	t.Run("ResourceAttributesEnv", func(t *testing.T) {
		t.Setenv("OTEL_SERVICE_NAME", "")
		t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.name=app-from-env")

		res, err := newResource(context.Background(), newOptions(WithServiceName("app")))
		require.NoError(t, err)

		require.Equal(t, "app-from-env", attributes(t, res)["service.name"])
	})
	t.Run("ResourceOptionsReplaced", func(t *testing.T) {
		t.Setenv("OTEL_SERVICE_NAME", "app-from-env")

		res, err := newResource(context.Background(), newOptions(
			WithResourceOptions(resource.WithTelemetrySDK()),
			WithServiceName("app"),
		))
		require.NoError(t, err)

		require.Equal(t, "app-from-env", attributes(t, res)["service.name"])
	})
}
