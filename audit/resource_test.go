package audit_test

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.uber.org/zap/zaptest"

	"github.com/go-faster/sdk/audit"
	"github.com/go-faster/sdk/zctx"
)

func testResource() *resource.Resource {
	return resource.NewSchemaless(
		semconv.ServiceName("auth-api"),
		semconv.ServiceNamespace("iam"),
		semconv.ServiceVersion("1.2.3"),
		semconv.DeploymentEnvironmentName("prod"),
		semconv.HostName("host-7"),
		semconv.DeviceManufacturer("GoFaster"),
	)
}

func TestWithResourceProvenance(t *testing.T) {
	ctx := zctx.Base(context.Background(), zaptest.NewLogger(t))
	exp := audit.NewMemoryExporter()
	r, _, err := audit.New(ctx,
		audit.WithExporter(exp),
		audit.WithResource(testResource()),
	)
	require.NoError(t, err)
	r.Emit(ctx, audit.NewEvent(audit.EventLogin, "alice", audit.ResultSuccess))
	got := exp.Events()
	require.Len(t, got, 1)
	require.Equal(t, "auth-api", got[0].Service)
	require.Equal(t, "iam", got[0].Component)
	require.Equal(t, "1.2.3", got[0].Version)
	require.Equal(t, "prod", got[0].Environment)
}

func TestWithResourceExplicitOverrides(t *testing.T) {
	ctx := zctx.Base(context.Background(), zaptest.NewLogger(t))
	exp := audit.NewMemoryExporter()
	r, _, err := audit.New(ctx,
		audit.WithExporter(exp),
		audit.WithResource(testResource()),
		audit.WithService("explicit-svc"),
	)
	require.NoError(t, err)
	r.Emit(ctx, audit.NewEvent(audit.EventLogin, "alice", audit.ResultSuccess))
	require.Equal(t, "explicit-svc", exp.Events()[0].Service)
}

func TestCEFWithResource(t *testing.T) {
	t.Run("Defaults", func(t *testing.T) {
		exp, err := audit.NewCEFExporter(
			audit.WithCEFResource(testResource()),
			audit.WithCEFTransport(audit.CEFOverWriter(io.Discard)),
		)
		require.NoError(t, err)
		require.Equal(t, "GoFaster", exp.DeviceVendor())
		require.Equal(t, "auth-api", exp.DeviceProduct())
		require.Equal(t, "1.2.3", exp.DeviceVersion())
	})
	t.Run("ExplicitWins", func(t *testing.T) {
		exp, err := audit.NewCEFExporter(
			audit.WithCEFResource(testResource()),
			audit.WithCEFDevice("V", "P", "9"),
			audit.WithCEFTransport(audit.CEFOverWriter(io.Discard)),
		)
		require.NoError(t, err)
		require.Equal(t, "V", exp.DeviceVendor())
		require.Equal(t, "P", exp.DeviceProduct())
		require.Equal(t, "9", exp.DeviceVersion())
	})
}

func TestSyslogWithResource(t *testing.T) {
	t.Run("Defaults", func(t *testing.T) {
		exp, err := audit.NewSyslogExporter("127.0.0.1:514",
			audit.WithSyslogResource(testResource()),
		)
		require.NoError(t, err)
		require.Equal(t, "host-7", exp.Hostname())
		require.Equal(t, "auth-api", exp.AppName())
	})
	t.Run("ExplicitWins", func(t *testing.T) {
		exp, err := audit.NewSyslogExporter("127.0.0.1:514",
			audit.WithSyslogResource(testResource()),
			audit.WithSyslogHostname("explicit"),
			audit.WithSyslogAppName("explicit-app"),
		)
		require.NoError(t, err)
		require.Equal(t, "explicit", exp.Hostname())
		require.Equal(t, "explicit-app", exp.AppName())
	})
}
