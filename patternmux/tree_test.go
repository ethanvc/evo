package patternmux

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTreeParamRoute(t *testing.T) {
	var tree Node[string]
	tree.addRoute("/abc/:user-id", "ok", routeMeta{
		raw:       "/abc/:user-id",
		canonical: "/abc/:user-id",
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
	require.Equal(t, "user-id", (*caps)[0].Key)
	require.Equal(t, "13455", (*caps)[0].Value)
}

func TestTreeCatchAllRoute(t *testing.T) {
	var tree Node[string]
	tree.addRoute("/abc/*path", "ok", routeMeta{
		raw:       "/abc/*path",
		canonical: "/abc/*path",
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
	require.Equal(t, "path", (*caps)[0].Key)
	require.Equal(t, "/a/b/c", (*caps)[0].Value)
}
