package patternmux

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNodeAccessors(t *testing.T) {
	n := Node[int]{
		raw:             "/abc/{replace::user-id;until-slash}",
		canonical:       "/abc/:user-id",
		hasKeep:         true,
		cachedConverted: "/abc/100",
	}

	require.Equal(t, "/abc/{replace::user-id;until-slash}", n.Raw())
	require.Equal(t, "/abc/:user-id", n.Canonical())
	require.True(t, n.HasKeep())
	require.Equal(t, "/abc/100", n.CachedConverted())
}
