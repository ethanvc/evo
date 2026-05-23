package patternmux

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLiteralOnly(t *testing.T) {
	segs, err := Parse("/abc/def")
	require.NoError(t, err)
	require.Equal(t, []Segment{Literal{Text: "/abc/def"}}, segs)
}

func TestParseReplaceUntilSlash(t *testing.T) {
	segs, err := Parse("/abc/{replace::user-id;until-slash}")
	require.NoError(t, err)
	require.Len(t, segs, 2)
	require.Equal(t, Literal{Text: "/abc/"}, segs[0])
	e := segs[1].(Expr)
	require.Equal(t, ActionReplace, e.Action)
	require.Equal(t, ":user-id", e.Name)
	require.Equal(t, []Rule{RuleUntilSlash}, e.Rules)
}

func TestParseReplaceRest(t *testing.T) {
	segs, err := Parse("/abc/{replace:*path;rest}")
	require.NoError(t, err)
	e := segs[1].(Expr)
	require.Equal(t, "*path", e.Name)
	require.Equal(t, []Rule{RuleRest}, e.Rules)
}

func TestParseKeepWithRules(t *testing.T) {
	segs, err := Parse("error code {keep;digit}")
	require.NoError(t, err)
	require.Len(t, segs, 2)
	e := segs[1].(Expr)
	require.Equal(t, ActionKeep, e.Action)
	require.Equal(t, "", e.Name, "unnamed keep has empty Name")
	require.Equal(t, []Rule{RuleDigit}, e.Rules)
}

func TestParseKeepWithName(t *testing.T) {
	segs, err := Parse("error code {keep:err-code;digit}")
	require.NoError(t, err)
	require.Len(t, segs, 2)
	e := segs[1].(Expr)
	require.Equal(t, ActionKeep, e.Action)
	require.Equal(t, "err-code", e.Name,
		"name follows the same `action:name` shape for keep as for replace")
	require.Equal(t, []Rule{RuleDigit}, e.Rules)
}

func TestParseKeepEmptyNameRejected(t *testing.T) {
	// `keep:` with empty name mirrors `replace:` rejection.
	_, err := Parse("error code {keep:;digit}")
	require.ErrorIs(t, err, ErrInvalidSyntax)
}

func TestParseMissingRule(t *testing.T) {
	_, err := Parse("/abc/{replace::id}")
	require.ErrorIs(t, err, ErrMissingRule)
}

func TestParseUnknownRule(t *testing.T) {
	_, err := Parse("/abc/{replace::id;unknown}")
	require.ErrorIs(t, err, ErrUnknownRule)
}

func TestParseUnclosedBrace(t *testing.T) {
	_, err := Parse("/abc/{replace::id;until-slash")
	require.ErrorIs(t, err, ErrInvalidSyntax)
}
