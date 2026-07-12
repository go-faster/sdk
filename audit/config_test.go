package audit_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/go-faster/sdk/audit"
	"github.com/go-faster/sdk/zctx"
)

func TestOptions(t *testing.T) {
	tests := []struct {
		options []audit.Option
		check   func(t *testing.T, e audit.Event)
	}{
		{
			[]audit.Option{
				audit.WithClock(func() time.Time { return fixedT0 }),
				audit.WithIDGenerator(func() string { return "test-event-id" }),
			},
			func(t *testing.T, e audit.Event) {
				require.Equal(t, fixedT0, e.Time)
				require.Equal(t, "test-event-id", e.ID)
			},
		},
		{
			[]audit.Option{
				audit.WithService("svc"),
				audit.WithComponent("api"),
				audit.WithEnvironment("prod"),
				audit.WithVersion("v2"),
			},
			func(t *testing.T, e audit.Event) {
				require.Equal(t, "svc", e.Service)
				require.Equal(t, "api", e.Component)
				require.Equal(t, "prod", e.Environment)
				require.Equal(t, "v2", e.Version)
			},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			exp := &memoryExporter{}
			opts := append(tt.options, audit.WithExporter(exp))
			ctx := zctx.Base(context.Background(), zaptest.NewLogger(t))
			r, _, err := audit.New(ctx, opts...)
			require.NoError(t, err)
			r.Emit(ctx, audit.NewEvent(audit.EventLogin, "actor", audit.ResultSuccess))
			require.Len(t, exp.events, 1)
			tt.check(t, exp.events[0])
		})
	}
}
