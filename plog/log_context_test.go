package plog

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_getCallerName(t *testing.T) {
	n := getCallerName(0)
	require.Equal(t, "Test_getCallerName", n)
}
