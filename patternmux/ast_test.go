package patternmux

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActionAndRuleConstants(t *testing.T) {
	require.Equal(t, Action("replace"), ActionReplace)
	require.Equal(t, Action("keep"), ActionKeep)
	require.Equal(t, Rule("until-slash"), RuleUntilSlash)
	require.Equal(t, Rule("rest"), RuleRest)
}
