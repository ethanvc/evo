package patternmux

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func compileMust(t *testing.T, raw string) ([]Segment, compiledPattern) {
	t.Helper()
	segs, err := Parse(raw)
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	return segs, cp
}

func metaFor(cp compiledPattern, order uint64) patternMeta {
	return patternMeta{
		raw:           cp.Raw,
		pattern:       cp.Pattern,
		hasKeep:       cp.HasKeep,
		literalChars:  cp.LiteralChars,
		registerOrder: order,
	}
}

func TestTreeUntilSlashRule(t *testing.T) {
	segs, cp := compileMust(t, "/abc/{replace::user-id;until-slash}")
	var tree Node[string]
	tree.addPattern(segs, "ok", metaFor(cp, 1))

	match, caps, ok := tree.matchInput("/abc/13455")
	require.True(t, ok)
	require.NotNil(t, match)
	require.Equal(t, "ok", match.Value())
	require.NotNil(t, caps)
	require.Equal(t, Captures{{Key: ":user-id", Value: "13455"}}, *caps)
}

func TestTreeRestRule(t *testing.T) {
	segs, cp := compileMust(t, "/abc/{replace:*path;rest}")
	var tree Node[string]
	tree.addPattern(segs, "ok", metaFor(cp, 1))

	match, caps, ok := tree.matchInput("/abc/a/b/c")
	require.True(t, ok)
	require.Equal(t, "ok", match.Value())
	require.Equal(t, Captures{{Key: "*path", Value: "a/b/c"}}, *caps)
}

func TestTreeRestRuleWithoutSlashLiteral(t *testing.T) {
	segs, cp := compileMust(t, "abc{replace:*tail;rest}")
	var tree Node[string]
	tree.addPattern(segs, "ok", metaFor(cp, 1))

	match, caps, ok := tree.matchInput("abcxyz")
	require.True(t, ok)
	require.Equal(t, "ok", match.Value())
	require.Equal(t, Captures{{Key: "*tail", Value: "xyz"}}, *caps)
}

func TestTreeUntilBlankRule(t *testing.T) {
	segs, cp := compileMust(t, "log {replace::msg;until-blank} end")
	var tree Node[string]
	tree.addPattern(segs, "ok", metaFor(cp, 1))

	match, caps, ok := tree.matchInput("log hello end")
	require.True(t, ok)
	require.Equal(t, "ok", match.Value())
	require.Equal(t, Captures{{Key: ":msg", Value: "hello"}}, *caps)

	_, _, ok = tree.matchInput("log hello world end")
	require.False(t, ok, "until-blank stops at first blank; tail mismatch must miss")
}

func TestTreeDigitClassRule(t *testing.T) {
	segs, cp := compileMust(t, "/abc/{replace::id;until-slash;digit}")
	var tree Node[string]
	tree.addPattern(segs, "ok", metaFor(cp, 1))

	match, caps, ok := tree.matchInput("/abc/12345")
	require.True(t, ok)
	require.Equal(t, Captures{{Key: ":id", Value: "12345"}}, *caps)
	_ = match

	_, _, ok = tree.matchInput("/abc/12a45")
	require.False(t, ok, "non-digit shrinks consumed segment; tail mismatch")
}

func TestTreeHexClassRule(t *testing.T) {
	segs, cp := compileMust(t, "/h/{replace::id;until-slash;hexdigit}")
	var tree Node[string]
	tree.addPattern(segs, "ok", metaFor(cp, 1))

	match, caps, ok := tree.matchInput("/h/deadBEEF")
	require.True(t, ok)
	require.Equal(t, "ok", match.Value())
	require.Equal(t, Captures{{Key: ":id", Value: "deadBEEF"}}, *caps)

	_, _, ok = tree.matchInput("/h/zzz")
	require.False(t, ok)
}

func TestTreeMultipleWildcardsBacktrack(t *testing.T) {
	segs1, cp1 := compileMust(t, "/u/{replace::id;until-slash;digit}/orders")
	segs2, cp2 := compileMust(t, "/u/{replace::name;until-slash}/profile")
	var tree Node[string]
	tree.addPattern(segs1, "orders", metaFor(cp1, 1))
	tree.addPattern(segs2, "profile", metaFor(cp2, 2))

	// Numeric id, /orders → first wildcard wins by spec.
	match, caps, ok := tree.matchInput("/u/123/orders")
	require.True(t, ok)
	require.Equal(t, "orders", match.Value())
	require.Equal(t, Captures{{Key: ":id", Value: "123"}}, *caps)

	// Same numeric id but /profile path → first wildcard's subtree fails on
	// "/orders" mismatch, backtrack to second wildcard.
	match, caps, ok = tree.matchInput("/u/123/profile")
	require.True(t, ok)
	require.Equal(t, "profile", match.Value())
	require.Equal(t, Captures{{Key: ":name", Value: "123"}}, *caps)

	// Non-numeric id only matches the second wildcard.
	match, caps, ok = tree.matchInput("/u/alice/profile")
	require.True(t, ok)
	require.Equal(t, "profile", match.Value())
	require.Equal(t, Captures{{Key: ":name", Value: "alice"}}, *caps)
}

func TestTreeStaticBeatsWildcardSibling(t *testing.T) {
	segs1, cp1 := compileMust(t, "/api/{replace::id;until-slash}")
	segs2, cp2 := compileMust(t, "/api/v1/users")
	var tree Node[string]
	tree.addPattern(segs1, "by-id", metaFor(cp1, 1))
	tree.addPattern(segs2, "static", metaFor(cp2, 2))

	// Static literal /v1/users dominates the wildcard branch.
	match, caps, ok := tree.matchInput("/api/v1/users")
	require.True(t, ok)
	require.Equal(t, "static", match.Value())
	require.Nil(t, caps)

	// Other inputs fall through to the wildcard.
	match, caps, ok = tree.matchInput("/api/abc")
	require.True(t, ok)
	require.Equal(t, "by-id", match.Value())
	require.Equal(t, Captures{{Key: ":id", Value: "abc"}}, *caps)
}
