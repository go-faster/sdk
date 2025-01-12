//go:build race

package race

import "testing"

func TestRaceOn(t *testing.T) {
	Skip(t)
	t.Fatal("Should be skipped")
}
