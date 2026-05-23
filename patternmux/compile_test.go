package patternmux

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompileReplaceUntilSlash(t *testing.T) {
	segs, err := Parse("/abc/{replace::user-id;until-slash}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.Equal(t, "/abc/{replace::user-id;until-slash}", cp.Raw)
	require.Equal(t, "/abc/:user-id", cp.Canonical)
	require.Equal(t, "/abc/:user-id", cp.Pattern)
	require.Equal(t, "/abc/:user-id", cp.CachedConverted)
	require.False(t, cp.HasKeep)
	require.Equal(t, len("/abc/"), cp.LiteralPrefix)
}

func TestCompileReplaceRest(t *testing.T) {
	segs, err := Parse("/abc/{replace:*path;rest}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.Equal(t, "/abc/*path", cp.Canonical)
	require.Equal(t, "/abc/*path", cp.CachedConverted)
	require.False(t, cp.HasKeep)
}

func TestCompileKeepHasKeepAndNoCachedConverted(t *testing.T) {
	segs, err := Parse("error code {keep;digit}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.True(t, cp.HasKeep)
	require.Equal(t, "error code {keep;digit}", cp.Canonical)
	require.Equal(t, "error code "+PlaceholderName, cp.Pattern,
		"keep is unnamed; Pattern substitutes PlaceholderName")
	require.Equal(t, "", cp.CachedConverted, "keep patterns build Converted at Lookup time")
}

func TestCompilePatternUnnamedReplaceUsesPlaceholder(t *testing.T) {
	segs, err := Parse("v={replace;digit}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.Equal(t, "v=", cp.Canonical, "unnamed replace contributes nothing to Canonical")
	require.Equal(t, "v="+PlaceholderName, cp.Pattern,
		"unnamed replace becomes PlaceholderName in Pattern")
}

func TestCompileKeepWithNamePropagates(t *testing.T) {
	segs, err := Parse("error code {keep:err-code;digit}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.True(t, cp.HasKeep)
	require.Equal(t, "error code {keep:err-code;digit}", cp.Canonical,
		"keep keeps full raw expression in Canonical (now including the name)")
	require.Equal(t, "error code err-code", cp.Pattern,
		"named keep uses its name in Pattern, just like named replace")
}

func TestCompileMultiRuleCanonicalUsesNameOnly(t *testing.T) {
	segs, err := Parse("/abc/{replace::id;until-slash;digit}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.Equal(t, "/abc/:id", cp.Canonical, "Canonical strips rule list, keeps replace name")
	require.Equal(t, "/abc/:id", cp.CachedConverted)
	require.False(t, cp.HasKeep)
}

func TestCompileRestWithoutSlashLiteral(t *testing.T) {
	segs, err := Parse("abc{replace:*tail;rest}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.Equal(t, "abc*tail", cp.Canonical)
	require.Equal(t, "abc*tail", cp.CachedConverted)
}

func TestCompileLiteralOnlyNoExpr(t *testing.T) {
	segs, err := Parse("/api/v1/users")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.Equal(t, "/api/v1/users", cp.Canonical)
	require.Equal(t, "/api/v1/users", cp.CachedConverted)
	require.Equal(t, len("/api/v1/users"), cp.LiteralPrefix)
	require.Len(t, segs, 1)
	_, isLiteral := segs[0].(Literal)
	require.True(t, isLiteral)
}
