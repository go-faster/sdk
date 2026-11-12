package audit_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-faster/sdk/audit"
)

func TestRedact(t *testing.T) {
	tests := []struct {
		redactor audit.Redactor
		check    func(t *testing.T, e audit.Event)
	}{
		{audit.MaskFields("ActorID", "token"), func(t *testing.T, e audit.Event) {
			require.Equal(t, "***", e.ActorID)
			require.Equal(t, "***", e.Attributes["token"])
		}},
		{audit.TruncateFields(3, "ActorID"), func(t *testing.T, e audit.Event) { require.Equal(t, "ali", e.ActorID) }},
		{audit.HashFields("ActorID"), func(t *testing.T, e audit.Event) { require.Len(t, e.ActorID, 64) }},
		{audit.ChainRedactors(audit.MaskFields("ActorID"), audit.TruncateFields(2, "ActorID")), func(t *testing.T, e audit.Event) {
			require.Equal(t, "**", e.ActorID)
		}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			e := minimalEvent()
			e.Attributes = map[string]string{"token": "secret"}
			tt.redactor.Redact(&e)
			tt.check(t, e)
		})
	}
}
