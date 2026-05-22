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
	require.Equal(t, "/abc/:user-id", cp.CachedConverted)
	require.False(t, cp.HasKeep)
	require.Equal(t, profileRoute, cp.Profile)
	require.Equal(t, "/abc/:user-id", cp.RoutePath)
}

func TestCompileReplaceRest(t *testing.T) {
	segs, err := Parse("/abc/{replace:*path;rest}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.Equal(t, "/abc/*path", cp.Canonical)
	require.Equal(t, "/abc/*path", cp.RoutePath)
}

func TestCompileKeepIsTextProfile(t *testing.T) {
	segs, err := Parse("error code {keep;digit}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.True(t, cp.HasKeep)
	require.Equal(t, profileText, cp.Profile)
	require.Equal(t, "error code {keep;digit}", cp.Canonical)
	require.Equal(t, "", cp.CachedConverted)
}

func TestCompileDigitRuleIsTextProfile(t *testing.T) {
	segs, err := Parse("/abc/{replace::id;until-slash;digit}")
	require.NoError(t, err)
	cp, err := Compile(segs)
	require.NoError(t, err)
	require.Equal(t, profileText, cp.Profile)
}
