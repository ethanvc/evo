package evohttp

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_splitPatternPath(t *testing.T) {
	f := splitPatternPathTest
	f(t, "/user/:user_id/detail", []string{"/user/", ":user_id", "/detail"})
	f(t, "/user/*path", []string{"/user/", "*path"})
	f(t, "/user/:id1/:id2/xxx", []string{"/user/", ":id1", "/", ":id2", "/xxx"})
}

func splitPatternPathTest(t *testing.T, s string, expect []string) {
	result := splitPatternPath(s)
	require.Equal(t, len(expect), len(result), s)
	for k, v := range result {
		require.Equal(t, expect[k], v)
	}
}

func Test_getCommonPrefixLength(t *testing.T) {
	require.Equal(t, 2, getCommonPrefixLength("nihao", "ni"))
	require.Equal(t, 0, getCommonPrefixLength("nihao", "0"))
}
