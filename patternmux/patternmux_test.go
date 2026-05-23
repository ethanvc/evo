package patternmux

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func requireLookupMiss[T any](t *testing.T, node *Node[T], caps *Captures, converted string, ok bool) {
	t.Helper()
	require.False(t, ok, "expected lookup miss")
	require.Nil(t, node)
	require.Nil(t, caps)
	require.Equal(t, "", converted, "converted must be empty on miss")
}

func requireMuxRegistered[T any](t *testing.T, mux *Mux[T], raw, canonical string, value T) {
	t.Helper()
	require.Contains(t, mux.byRaw, raw)
	require.Equal(t, value, mux.byCanonical[canonical])
}

func TestMuxRegisterLookupParamRoute(t *testing.T) {
	const (
		raw        = "/abc/{replace::user-id;until-slash}"
		canonical  = "/abc/:user-id"
		input      = "/abc/13455"
		wantValue  = "user-get"
	)
	mux := New[string]()
	require.NoError(t, mux.Register(raw, wantValue))
	requireMuxRegistered(t, mux, raw, canonical, wantValue)

	node, caps, converted, ok := mux.Lookup(input)
	require.True(t, ok)
	require.NotNil(t, node)
	require.Equal(t, wantValue, node.Value())
	require.Equal(t, raw, node.Raw())
	require.Equal(t, canonical, node.Canonical())
	require.False(t, node.HasKeep())
	require.Equal(t, canonical, node.CachedConverted())
	require.Equal(t, canonical, converted)
	require.Equal(t, node.CachedConverted(), converted)
	require.NotNil(t, caps)
	require.Equal(t, Captures{{Key: ":user-id", Value: "13455"}}, *caps)
	PutCaptures(caps)
}

func TestMuxRegisterLookupCatchAll(t *testing.T) {
	const (
		raw       = "/abc/{replace:*path;rest}"
		canonical = "/abc/*path"
		input     = "/abc/a/b/c"
		wantValue = "files"
	)
	mux := New[string]()
	require.NoError(t, mux.Register(raw, wantValue))
	requireMuxRegistered(t, mux, raw, canonical, wantValue)

	node, caps, converted, ok := mux.Lookup(input)
	require.True(t, ok)
	require.NotNil(t, node)
	require.Equal(t, wantValue, node.Value())
	require.Equal(t, raw, node.Raw())
	require.Equal(t, canonical, node.Canonical())
	require.False(t, node.HasKeep())
	require.Equal(t, canonical, node.CachedConverted())
	require.Equal(t, canonical, converted)
	require.NotNil(t, caps)
	require.Equal(t, Captures{{Key: "*path", Value: "a/b/c"}}, *caps)
	PutCaptures(caps)
}

func TestMuxLookupMiss(t *testing.T) {
	const (
		raw       = "/abc/{replace::user-id;until-slash}"
		canonical = "/abc/:user-id"
		wantValue = "v"
	)
	mux := New[string]()
	require.NoError(t, mux.Register(raw, wantValue))
	requireMuxRegistered(t, mux, raw, canonical, wantValue)

	node, caps, converted, ok := mux.Lookup("/other/1")
	requireLookupMiss(t, node, caps, converted, ok)

	// registered route path without param segment must not match
	node, caps, converted, ok = mux.Lookup("/abc/")
	requireLookupMiss(t, node, caps, converted, ok)
}

func TestMuxTextProfileRegisterOnly(t *testing.T) {
	const (
		raw       = "error code {keep;digit}"
		canonical = "error code {keep;digit}"
		wantValue = "log-parser"
	)
	mux := New[string]()
	require.NoError(t, mux.Register(raw, wantValue))
	requireMuxRegistered(t, mux, raw, canonical, wantValue)
	require.Equal(t, uint16(0), mux.maxCaptures, "text profile must not enter route tree")
	require.Equal(t, 1, mux.registerSeq)

	// v1: text profile is metadata-only; no match for full or partial input
	node, caps, converted, ok := mux.Lookup("error code 123456")
	requireLookupMiss(t, node, caps, converted, ok)

	node, caps, converted, ok = mux.Lookup("error code ")
	requireLookupMiss(t, node, caps, converted, ok)

	node, caps, converted, ok = mux.Lookup("error code")
	requireLookupMiss(t, node, caps, converted, ok)
}

func TestMuxDuplicateRaw(t *testing.T) {
	const (
		raw       = "/a/{replace::id;until-slash}"
		canonical = "/a/:id"
	)
	mux := New[int]()
	require.NoError(t, mux.Register(raw, 1))
	requireMuxRegistered(t, mux, raw, canonical, 1)

	err := mux.Register(raw, 2)
	require.ErrorIs(t, err, ErrDuplicatePattern)
	requireMuxRegistered(t, mux, raw, canonical, 1)
}

func TestMuxDuplicateCanonical(t *testing.T) {
	const (
		rawRoute = "/p/{replace::id;until-slash}"
		rawText  = "/p/{replace::id;until-slash;digit}"
		canonical = "/p/:id"
	)
	mux := New[int]()
	require.NoError(t, mux.Register(rawRoute, 1))
	requireMuxRegistered(t, mux, rawRoute, canonical, 1)

	// digit rule => text profile, but canonical still /p/:id
	err := mux.Register(rawText, 2)
	require.ErrorIs(t, err, ErrDuplicateCanonical)
	requireMuxRegistered(t, mux, rawRoute, canonical, 1)
	require.NotContains(t, mux.byRaw, rawText)
}

func TestMuxStaticOverParam(t *testing.T) {
	const (
		paramRaw       = "/api/{replace::id;until-slash}"
		paramCanonical = "/api/:id"
		staticRaw      = "/api/v1/users"
		staticCanonical = "/api/v1/users"
		input          = "/api/v1/users"
		wantValue      = "static-users"
	)
	mux := New[string]()
	require.NoError(t, mux.Register(paramRaw, "by-id"))
	require.NoError(t, mux.Register(staticRaw, wantValue))
	requireMuxRegistered(t, mux, paramRaw, paramCanonical, "by-id")
	requireMuxRegistered(t, mux, staticRaw, staticCanonical, wantValue)

	node, caps, converted, ok := mux.Lookup(input)
	require.True(t, ok)
	require.NotNil(t, node)
	require.Equal(t, wantValue, node.Value())
	require.Equal(t, staticRaw, node.Raw())
	require.Equal(t, staticCanonical, node.Canonical())
	require.False(t, node.HasKeep())
	require.Equal(t, staticCanonical, node.CachedConverted())
	require.Equal(t, staticCanonical, converted)
	require.Nil(t, caps)
}

func BenchmarkMuxParamRoute(b *testing.B) {
	mux := New[int]()
	_ = mux.Register("/api/v1/users/{replace::id;until-slash}", 1)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		node, caps, _, ok := mux.Lookup("/api/v1/users/12345")
		if !ok {
			b.Fatal("miss")
		}
		PutCaptures(caps)
		_ = node
	}
}
