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

func Test_extractTailPart(t *testing.T) {
	require.Equal(t, "b/c", extractTailPart("a/b/c"))
	require.Equal(t, "", extractTailPart(""))
	require.Equal(t, "/a", extractTailPart("/a"))
	require.Equal(t, "a/c", extractTailPart("/a/c"))
}
