package evohttp

import (
	"context"
	"github.com/ethanvc/evo/base"
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
	rootB := NewRouterBuilder()
	g := rootB.SubBuilder("/", HandlerFunc(hf))
	g.POSTF("/a/b/c", hf)
	item := rootB.router.ListAll()
	require.Equal(t, 1, len(item))
	require.Equal(t, 2, len(item[0].Node.Handlers))
}

func Test_3(t *testing.T) {
	rootB := NewRouterBuilder()

	rootB.POSTF("/api/users/register", hf)
	rootB.POSTF("/api/users/login", hf)

	// all below handler needs login state
	g := rootB.SubBuilder("/", HandlerFunc(hf))
	g.GET("/api/users/get-public-home-access-token", HandlerFunc(hf))

	g.POST("/api/weed-tasks/create", HandlerFunc(hf))
	g.POST("/api/weed-tasks/get", HandlerFunc(hf))
	data := []routeContent{
		{http.MethodPost, "/api/users/register"},
		{http.MethodPost, "/api/users/login"},
		{http.MethodGet, "/api/users/get-public-home-access-token"},
		{http.MethodPost, "/api/weed-tasks/create"},
		{http.MethodPost, "/api/weed-tasks/get"},
	}
	for _, d := range data {
		urlParam := make(map[string]string)
		n := rootB.router.Find(d.method, d.p, urlParam)
		require.Equal(t, d.p, n.FullPath)
	}
}

type routeContent struct {
	method string
	p      string
}

func hf(c context.Context, req any, info *RequestInfo, nexter base.Nexter[*RequestInfo]) (any, error) {
	return nil, nil
}
