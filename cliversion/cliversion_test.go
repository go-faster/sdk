package cliversion

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetInfo(t *testing.T) {
	info, ok := GetInfo("github.com/go-faster/sdk")
	require.True(t, ok)
	require.NotEmpty(t, info.GoVersion)
}

func TestInfoString(t *testing.T) {
	require.Equal(t, "version unknown "+runtime.GOOS+"/"+runtime.GOARCH, Info{}.String())
}
