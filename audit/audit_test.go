package audit_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-faster/errors"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/go-faster/sdk/audit"
	"github.com/go-faster/sdk/gold"
	"github.com/go-faster/sdk/zctx"
)

var fixedT0 = time.Date(2026, 7, 12, 14, 3, 22, 118000000, time.UTC)

func TestRecorder(t *testing.T) {
	tests := []struct {
		name func(t *testing.T, ctx context.Context)
	}{
		{func(t *testing.T, ctx context.Context) {
			one, two := audit.NewMemoryExporter(), audit.NewMemoryExporter()
			r, _, err := audit.New(ctx, audit.WithExporters(one, two))
			require.NoError(t, err)
			r.Record(ctx, minimalEvent())
			require.Len(t, one.Events(), 1)
			require.Len(t, two.Events(), 1)
		}},
		{func(t *testing.T, ctx context.Context) {
			exp := audit.NewMemoryExporter()
			r, _, err := audit.New(ctx, audit.WithExporter(exp), audit.WithRedactor(audit.MaskFields("ActorID")))
			require.NoError(t, err)
			r.Record(ctx, minimalEvent())
			require.Equal(t, "***", exp.Events()[0].ActorID)
		}},
		{func(t *testing.T, ctx context.Context) {
			r, _, err := audit.New(ctx, audit.WithExporter(&errorExporter{}))
			require.NoError(t, err)
			require.NotPanics(t, func() { r.Record(ctx, minimalEvent()) })
		}},
		{func(t *testing.T, ctx context.Context) {
			r := audit.NewNop()
			require.NotPanics(t, func() { r.Record(ctx, minimalEvent()) })
			require.NoError(t, r.Shutdown(ctx))
		}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			ctx := zctx.Base(context.Background(), zaptest.NewLogger(t))
			tt.name(t, ctx)
		})
	}
}

func TestEventBuilder(t *testing.T) {
	tests := []struct {
		fn func(t *testing.T, ctx context.Context)
	}{
		{func(t *testing.T, ctx context.Context) {
			_, err := audit.NewEvent("", "actor", audit.ResultSuccess).Build(ctx)
			require.Error(t, err)
		}},
		{func(t *testing.T, ctx context.Context) {
			exp := audit.NewMemoryExporter()
			r, _, err := audit.New(ctx,
				audit.WithExporter(exp),
				audit.WithClock(func() time.Time { return fixedT0 }),
				audit.WithIDGenerator(func() string { return "test-event-id" }),
				audit.WithService("svc"),
			)
			require.NoError(t, err)
			r.Emit(ctx, audit.NewEvent(audit.EventLogin, "actor", audit.ResultSuccess))
			require.Equal(t, "test-event-id", exp.Events()[0].ID)
			require.Equal(t, fixedT0, exp.Events()[0].Time)
			require.Equal(t, "svc", exp.Events()[0].Service)
		}},
		{func(t *testing.T, ctx context.Context) {
			tid, err := trace.TraceIDFromHex("00112233445566778899aabbccddeeff")
			require.NoError(t, err)
			sid, err := trace.SpanIDFromHex("0011223344556677")
			require.NoError(t, err)
			spanCtx := trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
				TraceID: tid, SpanID: sid, TraceFlags: trace.FlagsSampled,
			}))

			t.Run("NoEnrichment", func(t *testing.T) {
				exp := audit.NewMemoryExporter()
				r, _, err := audit.New(spanCtx, audit.WithExporter(exp))
				require.NoError(t, err)
				r.Record(spanCtx, minimalEvent())
				require.Empty(t, exp.Events()[0].TraceID)
				require.Empty(t, exp.Events()[0].SpanID)
				require.Empty(t, exp.Events()[0].CorrelationID)
			})

			t.Run("TraceEnrichment", func(t *testing.T) {
				exp := audit.NewMemoryExporter()
				r, _, err := audit.New(spanCtx, audit.WithExporter(exp), audit.WithTraceEnrichment())
				require.NoError(t, err)
				r.Record(spanCtx, minimalEvent())
				require.Equal(t, tid.String(), exp.Events()[0].TraceID)
				require.Equal(t, sid.String(), exp.Events()[0].SpanID)
			})

			t.Run("TraceEnrichmentNoSpan", func(t *testing.T) {
				exp := audit.NewMemoryExporter()
				r, _, err := audit.New(ctx, audit.WithExporter(exp), audit.WithTraceEnrichment())
				require.NoError(t, err)
				r.Record(ctx, minimalEvent())
				require.Empty(t, exp.Events()[0].TraceID)
				require.Empty(t, exp.Events()[0].SpanID)
			})

			t.Run("TraceEnrichmentPreservesExplicit", func(t *testing.T) {
				exp := audit.NewMemoryExporter()
				r, _, err := audit.New(spanCtx, audit.WithExporter(exp), audit.WithTraceEnrichment())
				require.NoError(t, err)
				e := minimalEvent()
				e.TraceID = "explicit-trace"
				e.SpanID = "explicit-span"
				r.Record(spanCtx, e)
				require.Equal(t, "explicit-trace", exp.Events()[0].TraceID)
				require.Equal(t, "explicit-span", exp.Events()[0].SpanID)
			})
		}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			tt.fn(t, zctx.Base(context.Background(), zaptest.NewLogger(t)))
		})
	}
}

func TestMain(m *testing.M) {
	gold.Init()
	os.Exit(m.Run())
}

type errorExporter struct{}

func (e *errorExporter) Export(ctx context.Context, events []audit.Event) error {
	return errors.New("boom")
}
func (e *errorExporter) Close(ctx context.Context) error { return nil }
func (e *errorExporter) Name() string                    { return "error" }

func minimalEvent() audit.Event {
	return audit.Event{
		ID:            "test-event-id",
		SchemaVersion: audit.CurrentSchemaVersion,
		Type:          audit.EventLogin,
		Action:        "login",
		Time:          fixedT0,
		ActorID:       "alice",
		ActorType:     audit.ActorUser,
		Result:        audit.ResultSuccess,
		Severity:      audit.SeverityMedium,
		Message:       "User login",
		Service:       "auth",
		Component:     "oidc",
		Environment:   "test",
		Version:       "v1",
	}
}

var _ = zap.String
