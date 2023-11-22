package evolog

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_extractTailPart(t *testing.T) {
	require.Equal(t, "b/c", extractTailPart("a/b/c"))
	require.Equal(t, "", extractTailPart(""))
	require.Equal(t, "/a", extractTailPart("/a"))
	require.Equal(t, "a/c", extractTailPart("/a/c"))
}
