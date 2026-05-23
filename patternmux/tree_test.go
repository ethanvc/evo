package patternmux

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTreeParamRoute(t *testing.T) {
	segs, err := Parse("/abc/{replace::user-id;until-slash}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)

	var tree Node[string]
	tree.addRoute(cp.RoutePath, "ok", routeMeta{
		raw:       cp.Raw,
		canonical: cp.Canonical,
	})

	match, caps, ok := tree.getValue("/abc/13455", func() *Captures {
		cs := make(Captures, 0, 1)
		return &cs
	})
	require.True(t, ok)
	require.NotNil(t, match)
	require.Equal(t, "ok", match.Value())
	require.NotNil(t, caps)
	require.Len(t, *caps, 1)
	require.Equal(t, ":user-id", (*caps)[0].Key)
	require.Equal(t, "13455", (*caps)[0].Value)
}

func TestTreeCatchAllRoute(t *testing.T) {
	segs, err := Parse("/abc/{replace:*path;rest}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)

	var tree Node[string]
	tree.addRoute(cp.RoutePath, "ok", routeMeta{
		raw:       cp.Raw,
		canonical: cp.Canonical,
	})

	match, caps, ok := tree.getValue("/abc/a/b/c", func() *Captures {
		cs := make(Captures, 0, 1)
		return &cs
	})
	require.True(t, ok)
	require.NotNil(t, match)
	require.Equal(t, "ok", match.Value())
	require.NotNil(t, caps)
	require.Len(t, *caps, 1)
	require.Equal(t, "*path", (*caps)[0].Key)
	require.Equal(t, "/a/b/c", (*caps)[0].Value)
}
