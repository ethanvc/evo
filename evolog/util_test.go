package evolog

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestGetPC(t *testing.T) {
	loc := GetCallerLocation(GetPC(0))
	require.Truef(t, strings.HasPrefix(loc, "util_test.go:"), loc)
}
