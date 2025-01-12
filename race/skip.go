package race

import "testing"

// Skip if race enabled.
func Skip(t *testing.T) {
	t.Helper()
	if Enabled {
		t.Skip("Skipping: -race enabled")
	}
}
