package autoenv

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdditionalEndpoints(t *testing.T) {
	t.Run("Unset", func(t *testing.T) {
		t.Setenv("GOFASTER_OTLP_ENDPOINTS", "")
		t.Setenv("GOFASTER_OTLP_TRACES_ENDPOINTS", "")
		require.Empty(t, AdditionalEndpoints("TRACES"))
	})
	t.Run("Common", func(t *testing.T) {
		t.Setenv("GOFASTER_OTLP_ENDPOINTS", "http://a:4317, http://b:4317 ,,http://a:4317")
		require.Equal(t, []string{"http://a:4317", "http://b:4317"}, AdditionalEndpoints("TRACES"))
	})
	t.Run("SignalOverride", func(t *testing.T) {
		t.Setenv("GOFASTER_OTLP_ENDPOINTS", "http://a:4317")
		t.Setenv("GOFASTER_OTLP_TRACES_ENDPOINTS", "http://b:4317")
		require.Equal(t, []string{"http://b:4317"}, AdditionalEndpoints("TRACES"))
		require.Equal(t, []string{"http://a:4317"}, AdditionalEndpoints("LOGS"))
	})
}

func TestParseExporters(t *testing.T) {
	tests := []struct {
		value   string
		want    []string
		wantErr bool
	}{
		{"otlp", []string{"otlp"}, false},
		{" otlp ", []string{"otlp"}, false},
		{"otlp,console", []string{"otlp", "console"}, false},
		{"otlp, console ,", []string{"otlp", "console"}, false},
		{"otlp,otlp", []string{"otlp"}, false},
		{"none", []string{"none"}, false},

		{"", nil, true},
		{",", nil, true},
		{"none,otlp", nil, true},
		{"otlp,none", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, err := ParseExporters(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
