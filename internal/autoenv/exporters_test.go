package autoenv

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
