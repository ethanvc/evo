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
	cs := newCaptures()
	*cs = append(*cs, Capture{Key: "a", Value: "b"})
	PutCaptures(cs)
	cs2 := newCaptures()
	require.Empty(t, *cs2)
	require.GreaterOrEqual(t, cap(*cs2), defaultCapturesCap, "pooled slice keeps default capacity")
}

func TestCapturesPoolDoesNotShrinkRetained(t *testing.T) {
	// When a captures slice grows past the default cap during use, returning
	// it to the pool must NOT reset the backing array down to the default.
	// Otherwise subsequent users of the same slot would re-grow on every reuse.
	cs := newCaptures()
	for i := 0; i < defaultCapturesCap+4; i++ {
		*cs = append(*cs, Capture{Key: "k", Value: "v"})
	}
	grownCap := cap(*cs)
	require.Greater(t, grownCap, defaultCapturesCap)
	PutCaptures(cs)
	// The same slice header is still reachable via cs (caller violated contract
	// but the test verifies the pool didn't reallocate inside putCaptures).
	require.Equal(t, grownCap, cap(*cs))
	require.Empty(t, *cs)
}
