package audit_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-faster/sdk/audit"
	"github.com/go-faster/sdk/gold"
)

func TestEncodeCEF(t *testing.T) {
	tests := []struct {
		name string
		e    audit.Event
		cfg  audit.CEFEncodeConfig
	}{
		{"minimal", minimalEvent(), cefConfig()},
		{"full", fullEvent(), cefConfig()},
		{"escaping", escapingEvent(), cefConfig()},
		{"trailing_space", trailingSpaceEvent(), cefConfig()},
		{"custom_severity", minimalEvent(), audit.CEFEncodeConfig{DeviceVendor: "MyCompany", DeviceProduct: "AuthService", DeviceVersion: "2.0", SeverityMapper: func(audit.Severity) (string, error) { return "5", nil }}},
		{"ref_example", refExampleEvent(), audit.CEFEncodeConfig{DeviceVendor: "MyCompany", DeviceProduct: "AuthService", DeviceVersion: "2.0", SeverityMapper: func(audit.Severity) (string, error) { return "5", nil }}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			out, err := audit.EncodeCEF(tt.e, tt.cfg)
			require.NoError(t, err)
			gold.Str(t, string(out), "cef", tt.name+".txt")
		})
	}
}

func cefConfig() audit.CEFEncodeConfig {
	return audit.CEFEncodeConfig{DeviceVendor: "GoFaster", DeviceProduct: "SDK", DeviceVersion: "v1"}
}
