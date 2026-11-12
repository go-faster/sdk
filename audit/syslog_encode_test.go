package audit_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-faster/sdk/audit"
	"github.com/go-faster/sdk/gold"
)

func TestEncodeRFC5424(t *testing.T) {
	tests := []struct {
		name string
		e    audit.Event
		cfg  audit.SyslogEncodeConfig
	}{
		{"minimal", minimalEvent(), syslogConfig()},
		{"full", fullEvent(), syslogConfig()},
		{"escaping", escapingEvent(), syslogConfig()},
		{"nilvalue", minimalEvent(), audit.SyslogEncodeConfig{Facility: audit.LOG_AUTH, SDID: "audit@8734"}},
		{"custom_sdid", minimalEvent(), audit.SyslogEncodeConfig{Facility: audit.LOG_AUTH, Hostname: "host", AppName: "auth", ProcID: "123", SDID: "myapp@9999"}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			out, err := audit.EncodeRFC5424(tt.e, tt.cfg)
			require.NoError(t, err)
			gold.Str(t, string(out), "syslog", tt.name+".txt")
		})
	}
}

func syslogConfig() audit.SyslogEncodeConfig {
	return audit.SyslogEncodeConfig{Facility: audit.LOG_AUTH, Hostname: "host", AppName: "auth", ProcID: "123", SDID: "audit@8734"}
}
