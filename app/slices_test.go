package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_include(t *testing.T) {
	require.Equal(t, []int{1, 2, 3}, include([]int{1, 2}, 3))
}
