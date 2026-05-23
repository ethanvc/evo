package patternmux

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTreeUntilSlashRule(t *testing.T) {
	segs, err := Parse("/abc/{replace::user-id;until-slash}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)

	var tree Node[string]
	tree.insertIndexed(cp.IndexKey, "ok", patternMeta{
		raw:       cp.Raw,
		canonical: cp.Canonical,
	})

	match, caps, ok := tree.matchInput("/abc/13455", func() *Captures {
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

func TestTreeRestRule(t *testing.T) {
	segs, err := Parse("/abc/{replace:*path;rest}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)

	var tree Node[string]
	tree.insertIndexed(cp.IndexKey, "ok", patternMeta{
		raw:       cp.Raw,
		canonical: cp.Canonical,
	})

	match, caps, ok := tree.matchInput("/abc/a/b/c", func() *Captures {
		cs := make(Captures, 0, 1)
		return &cs
	})
	require.True(t, ok)
	require.NotNil(t, match)
	require.Equal(t, "ok", match.Value())
	require.NotNil(t, caps)
	require.Len(t, *caps, 1)
	require.Equal(t, "*path", (*caps)[0].Key)
	require.Equal(t, "a/b/c", (*caps)[0].Value)
}

func TestTreeRestRuleWithoutSlashLiteral(t *testing.T) {
	segs, err := Parse("abc{replace:*tail;rest}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)

	var tree Node[string]
	tree.insertIndexed(cp.IndexKey, "ok", patternMeta{
		raw:       cp.Raw,
		canonical: cp.Canonical,
	})

	match, caps, ok := tree.matchInput("abcxyz", func() *Captures {
		cs := make(Captures, 0, 1)
		return &cs
	})
	require.True(t, ok)
	require.Equal(t, "ok", match.Value())
	require.Equal(t, Captures{{Key: "*tail", Value: "xyz"}}, *caps)
}
