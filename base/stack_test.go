package base

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetCallerFrame(t *testing.T) {
	f := GetCallerFrame(GetPc(0))
	require.Equal(t, "github.com/ethanvc/evo/base.TestGetCallerFrame", f.Function)
}
