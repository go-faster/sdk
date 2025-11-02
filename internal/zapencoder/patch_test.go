package zapencoder_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-faster/sdk/gold"
	"github.com/go-faster/sdk/internal/zapencoder"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func testEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		NameKey:        "name",
		TimeKey:        "ts",
		CallerKey:      "caller",
		FunctionKey:    "func",
		StacktraceKey:  "stacktrace",
		LineEnding:     "\n",
		EncodeTime:     zapcore.EpochTimeEncoder,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

func constantTimeEncoder(now time.Time) zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		encodeTimeLayout(now, time.RFC3339Nano, enc)
	}
}

func encodeTimeLayout(t time.Time, layout string, enc zapcore.PrimitiveArrayEncoder) {
	type appendTimeEncoder interface {
		AppendTimeLayout(time.Time, string)
	}

	if enc, ok := enc.(appendTimeEncoder); ok {
		enc.AppendTimeLayout(t, layout)
		return
	}

	enc.AppendString(t.Format(layout))
}

func TestEncoder(t *testing.T) {
	const encoderName = "zapencoder"
	err := zap.RegisterEncoder(encoderName, func(config zapcore.EncoderConfig) (zapcore.Encoder, error) {
		enc := zapencoder.New(config)
		return enc, nil
	})
	require.NoError(t, err)

	// open a file, and hold reference to the close
	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.jsonl")

	writer, closeFile, err := zap.Open(outPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		closeFile()
	})

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = constantTimeEncoder(time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC))

	core := zapencoder.NewCustomCore(zapencoder.New(encoderConfig), writer, zap.NewAtomicLevel())
	lg := zap.New(core)

	ctx := t.Context()
	lg.With(
		zap.Any("ctx", ctx),
	).Info("With context")
	lg.Info("With context",
		zap.Any("ctx", ctx),
	)

	ctx = trace.ContextWithSpanContext(ctx,
		trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: [16]byte{0x4b, 0xf9, 0x2f, 0x35, 0x77, 0xb3, 0x4d, 0xa6, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36},
			SpanID:  [8]byte{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7},
		}),
	)

	lg.Info("With span",
		zap.Any("ctx", ctx),
	)

	require.NoError(t, lg.Sync())

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)

	t.Logf("%s", data)

	gold.Str(t, string(data), "logs.jsonl")
}
