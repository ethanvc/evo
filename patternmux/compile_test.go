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
	require.Equal(t, "/abc/:user-id", cp.Pattern)
	require.False(t, cp.HasKeep)
	require.Equal(t, len("/abc/"), cp.LiteralChars)
}

func TestCompileReplaceRest(t *testing.T) {
	segs, err := Parse("/abc/{replace:*path;rest}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.Equal(t, "/abc/*path", cp.Pattern)
	require.False(t, cp.HasKeep)
}

func TestCompileKeepHasKeep(t *testing.T) {
	segs, err := Parse("error code {keep;digit}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.True(t, cp.HasKeep)
	require.Equal(t, "error code "+PlaceholderName, cp.Pattern,
		"keep is unnamed; Pattern substitutes PlaceholderName")
}

func TestCompilePatternUnnamedReplaceUsesPlaceholder(t *testing.T) {
	segs, err := Parse("v={replace;digit}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.Equal(t, "v="+PlaceholderName, cp.Pattern,
		"unnamed replace becomes PlaceholderName in Pattern")
}

func TestCompileKeepWithNamePropagates(t *testing.T) {
	segs, err := Parse("error code {keep:err-code;digit}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.True(t, cp.HasKeep)
	require.Equal(t, "error code err-code", cp.Pattern,
		"named keep uses its name in Pattern, just like named replace")
}

func TestCompileMultiRulePatternUsesNameOnly(t *testing.T) {
	segs, err := Parse("/abc/{replace::id;until-slash;digit}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.Equal(t, "/abc/:id", cp.Pattern, "Pattern strips rule list, keeps replace name")
	require.False(t, cp.HasKeep)
}

func TestCompileRestWithoutSlashLiteral(t *testing.T) {
	segs, err := Parse("abc{replace:*tail;rest}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.Equal(t, "abc*tail", cp.Pattern)
}

func TestCompileLiteralOnlyNoExpr(t *testing.T) {
	segs, err := Parse("/api/v1/users")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.Equal(t, "/api/v1/users", cp.Pattern)
	require.Equal(t, len("/api/v1/users"), cp.LiteralChars)
	require.Len(t, segs, 1)
	_, isLiteral := segs[0].(Literal)
	require.True(t, isLiteral)
}
