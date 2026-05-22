package patternmux

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCapturesByName(t *testing.T) {
	cs := Captures{
		{Key: "user-id", Value: "13455"},
		{Key: "", Value: "123456"},
	}
	require.Equal(t, "13455", cs.ByName("user-id"))
	require.Equal(t, "", cs.ByName("missing"))
}

func TestCapturesPoolReuse(t *testing.T) {
	cs := newCaptures(4)
	*cs = append(*cs, Capture{Key: "a", Value: "b"})
	PutCaptures(cs)
	cs2 := newCaptures(4)
	require.Empty(t, *cs2)
}
