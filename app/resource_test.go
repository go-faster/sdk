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
