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
	osArch := runtime.GOOS + "/" + runtime.GOARCH
	require.Equal(t, "version unknown "+osArch, Info{}.String())

	info := Info{
		Version: "v1.2.3",
		Commit:  "abcdef0123456789",
	}
	require.Equal(t, "version v1.2.3-abcdef012345 "+osArch, info.String())

	info.Modified = true
	require.Equal(t, "version v1.2.3-abcdef012345-dirty "+osArch, info.String())
}
