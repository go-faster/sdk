package zctx

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func newTestTracer() trace.Tracer {
	exporter := tracetest.NewInMemoryExporter()
	randSource := rand.NewSource(15)
	tp := tracesdk.NewTracerProvider(
		// Using deterministic random ids.
		tracesdk.WithIDGenerator(&randomIDGenerator{
			rand: rand.New(randSource),
		}),
		tracesdk.WithBatcher(exporter,
			tracesdk.WithBatchTimeout(0), // instant
		),
	)
	return tp.Tracer("test")
}

func assertEmpty(t testing.TB, logs *observer.ObservedLogs) {
	t.Helper()
	assert.Equal(t, 0, logs.Len(), "Expected empty ObservedLogs to have zero length.")
	assert.Equal(t, []observer.LoggedEntry{}, logs.All(), "Unexpected LoggedEntries in empty ObservedLogs.")
}

func assertEntries(t testing.TB, logs *observer.ObservedLogs, want ...observer.LoggedEntry) {
	t.Helper()

	all := logs.TakeAll()
	for i := range all {
		all[i].Time = time.Time{}
	}

	assert.Equal(t, len(want), len(all), "Unexpected observed logs Len.")

	for i := 0; i < len(want); i++ {
		b, a := all[i], want[i]
		assert.Equalf(t, a.Message, b.Message, "[%d]: Unexpected message.", i)
		assert.Equalf(t, a.Level, b.Level, "[%d]: Unexpected level.", i)

		if assert.Equalf(t, len(a.Context), len(b.Context), "[%d]: Unexpected context length.", i) {
			expectedFields := make(map[string]zap.Field, len(a.Context))
			haveFields := make(map[string]zap.Field, len(b.Context))
			for j := 0; j < len(a.Context); j++ {
				expectedFields[a.Context[j].Key] = a.Context[j]
			}
			for j := 0; j < len(b.Context); j++ {
				haveFields[b.Context[j].Key] = b.Context[j]
			}
			for k, v := range expectedFields {
				if _, ok := haveFields[k]; !ok {
					t.Errorf("[%d]: Missing field %q.", i, k)
					continue
				}
				af, hf := v, haveFields[k]
				assert.Equalf(t, af.Key, hf.Key, "[%d][%s]: Unexpected context key.", i, k)
				if aCtx, aOk := af.Interface.(SpanCompare); aOk {
					hCtx, hOk := hf.Interface.(context.Context)
					assert.Truef(t, hOk, "[%d][%s]: Unexpected context value.", i, k)
					hS := trace.SpanContextFromContext(hCtx)
					assert.Truef(t, aCtx.Equal(hS), "[%d][%s]: Unexpected span context. (%s != %s-%s)",
						i, k, aCtx, hS.TraceID(), hS.SpanID(),
					)
				} else {
					assert.Equalf(t, af.Type, hf.Type, "[%d][%s]: Unexpected context type.", i, k)
					assert.Equalf(t, af.Interface, hf.Interface, "[%d][%s]: Unexpected context value.", i, k)
				}
				assert.Equalf(t, af.String, hf.String, "[%d][%s]: Unexpected context value.", i, k)
			}
		}
	}
}

type randomIDGenerator struct {
	sync.Mutex
	rand *rand.Rand
}

// NewSpanID returns a non-zero span ID from a randomly-chosen sequence.
func (gen *randomIDGenerator) NewSpanID(_ context.Context, _ trace.TraceID) (sid trace.SpanID) {
	gen.Lock()
	defer gen.Unlock()
	gen.rand.Read(sid[:])
	return sid
}

// NewIDs returns a non-zero trace ID and a non-zero span ID from a
// randomly-chosen sequence.
func (gen *randomIDGenerator) NewIDs(_ context.Context) (tid trace.TraceID, sid trace.SpanID) {
	gen.Lock()
	defer gen.Unlock()
	gen.rand.Read(tid[:])
	gen.rand.Read(sid[:])
	return tid, sid
}

func do(ctx context.Context, tracer trace.Tracer, depth int) {
	ctx, span := tracer.Start(ctx, fmt.Sprintf("do(%d)", depth))
	From(ctx).Info("do", zap.Int("depth", depth))
	if depth > 0 {
		do(ctx, tracer, depth-1)
	}
	defer span.End()
}

func TestFrom(t *testing.T) {
	obs, logs := observer.New(zap.DebugLevel)
	assertEmpty(t, logs)

	assert.NoError(t, obs.Sync(), "Unexpected failure in no-op Sync")

	lg := zap.New(obs).With(zap.Int("i", 1))
	lg.Info("foo")

	assertEntries(t, logs, observer.LoggedEntry{
		Entry:   zapcore.Entry{Level: zap.InfoLevel, Message: "foo"},
		Context: []zapcore.Field{zap.Int("i", 1)},
	})

	ctx := Base(context.Background(), lg)
	From(ctx).Info("baz")
	assertEntries(t, logs, observer.LoggedEntry{
		Entry:   zapcore.Entry{Level: zap.InfoLevel, Message: "baz"},
		Context: []zapcore.Field{zap.Int("i", 1)},
	})

	ctx = With(ctx, zap.Int("j", 2))
	From(ctx).Info("baz")
	assertEntries(t, logs, observer.LoggedEntry{
		Entry:   zapcore.Entry{Level: zap.InfoLevel, Message: "baz"},
		Context: []zapcore.Field{zap.Int("i", 1), zap.Int("j", 2)},
	})

	tracer := newTestTracer()
	do(ctx, tracer, 3)
	want := []observer.LoggedEntry{
		{
			Entry: zapcore.Entry{Level: zap.InfoLevel, Message: "do"},
			Context: []zapcore.Field{
				zap.Int("depth", 3),
				zap.String("trace_id", "47058b76ab7d2a10a2ef6534312d205a"),
				zap.String("span_id", "aa1a08609e5aacf2"),
				zap.Int("i", 1), zap.Int("j", 2),
			},
		},
		{
			Entry: zapcore.Entry{Level: zap.InfoLevel, Message: "do"},
			Context: []zapcore.Field{
				zap.Int("depth", 2),
				zap.String("trace_id", "47058b76ab7d2a10a2ef6534312d205a"),
				zap.String("span_id", "572a3c21b660fc50"),
				zap.Int("i", 1), zap.Int("j", 2),
			},
		},
		{
			Entry: zapcore.Entry{Level: zap.InfoLevel, Message: "do"},
			Context: []zapcore.Field{
				zap.Int("depth", 1),
				zap.String("trace_id", "47058b76ab7d2a10a2ef6534312d205a"),
				zap.String("span_id", "07b95cb1be0ea6cd"),
				zap.Int("i", 1), zap.Int("j", 2),
			},
		},
		{
			Entry: zapcore.Entry{Level: zap.InfoLevel, Message: "do"},
			Context: []zapcore.Field{
				zap.Int("depth", 4),
				zap.String("trace_id", "47058b76ab7d2a10a2ef6534312d205a"),
				zap.String("span_id", "6f539157d0433b08"),
				zap.Int("i", 1), zap.Int("j", 2),
			},
		},
	}
	assertEntries(t, logs, want...)
}

type SpanCompare struct {
	TraceID string
	SpanID  string
}

type SpanComparator interface {
	Equal(trace.SpanContext) bool
}

func newSpanComparator(traceID, spanID string) zap.Field {
	return zap.Any("ctx", SpanComparator(SpanCompare{
		TraceID: traceID,
		SpanID:  spanID,
	}))
}

func (c SpanCompare) Equal(sc trace.SpanContext) bool {
	if c.TraceID != sc.TraceID().String() {
		return false
	}
	if c.SpanID != sc.SpanID().String() {
		return false
	}
	return true
}
