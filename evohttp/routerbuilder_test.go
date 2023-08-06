package evohttp

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// https://docs.github.com/en/rest/commits/commits?apiVersion=2022-11-28
func TestRouterBuilder1(t *testing.T) {
	b := NewRouterBuilder()
	b.GETF("/repos/:OWNER/:REPO/commits", hf)
	b.GETF("/repos/:OWNER/:REPO/commits/:COMMIT_SHA/branches-where-head", hf)
	b.GETF("/static/*path", hf)
	items := b.router.ListAll()
	require.Equal(t, 3, len(items))
	params := make(map[string]string)
	n := b.router.Find(http.MethodGet, "/repos/xx/repo/commits", params)
	require.NotNil(t, n)
	require.Equal(t, 2, len(params))
	params = make(map[string]string)
	n = b.router.Find(http.MethodGet, "/static/a/b/c", params)
	require.NotNil(t, n)
	require.Equal(t, 1, len(params))
}

func Test_1(t *testing.T) {
	b := NewRouterBuilder()
	b.POSTF("/api/a", hf)
	b.POSTF("/api/b", hf)
	items := b.router.ListAll()
	require.Equal(t, 2, len(items))
}

func Test_2(t *testing.T) {
	b := NewRouterBuilder()
	g := b.SubBuilder("/", HandlerFunc(hf))
	g.POSTF("/a/b/c", hf)
	item := b.router.ListAll()
	require.Equal(t, 1, len(item))
	require.Equal(t, 2, len(item[0].Handlers))
}

func hf(c context.Context, req any, info *RequestInfo) (any, error) {
	return nil, nil
}
