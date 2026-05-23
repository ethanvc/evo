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

func requireMuxRegistered[T any](t *testing.T, mux *Mux[T], raw, canonical string, _ T) {
	t.Helper()
	require.Contains(t, mux.byRaw, raw)
	require.Contains(t, mux.byCanonical, canonical)
}

func TestMuxRegisterLookupUntilSlash(t *testing.T) {
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

func TestMuxRegisterLookupRest(t *testing.T) {
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

	// pattern ends at literal without a matching wildcard segment
	node, caps, converted, ok = mux.Lookup("/abc/")
	requireLookupMiss(t, node, caps, converted, ok)
}

func TestMuxScanKeepDigit(t *testing.T) {
	const (
		raw       = "error code {keep;digit}"
		canonical = "error code {keep;digit}"
		wantValue = "log-parser"
	)
	mux := New[string]()
	require.NoError(t, mux.Register(raw, wantValue))
	requireMuxRegistered(t, mux, raw, canonical, wantValue)
	require.Equal(t, 1, mux.registerSeq)

	node, caps, converted, ok := mux.Lookup("error code 123456")
	require.True(t, ok)
	require.NotNil(t, node)
	require.Equal(t, wantValue, node.Value())
	require.Equal(t, raw, node.Raw())
	require.Equal(t, canonical, node.Canonical())
	require.True(t, node.HasKeep())
	require.Equal(t, "error code 123456", converted)
	require.Equal(t, Captures{{Key: "", Value: "123456"}}, *caps)
	PutCaptures(caps)

	node, caps, converted, ok = mux.Lookup("error code ")
	requireLookupMiss(t, node, caps, converted, ok)

	// no trailing extra input allowed without a rest expression
	node, caps, converted, ok = mux.Lookup("error code 123 extra")
	requireLookupMiss(t, node, caps, converted, ok)
}

func TestMuxScanKeepAndReplaceMixed(t *testing.T) {
	const (
		raw   = "error code {keep;digit}, transaction-id is {replace;hexdigit}"
		input = "error code 123456, transaction-id is 123456abcd"
	)
	mux := New[string]()
	require.NoError(t, mux.Register(raw, "v"))

	node, caps, converted, ok := mux.Lookup(input)
	require.True(t, ok)
	require.True(t, node.HasKeep())
	require.Equal(t, "error code 123456, transaction-id is ", converted)
	require.Equal(t, Captures{
		{Key: "", Value: "123456"},
		{Key: "", Value: "123456abcd"},
	}, *caps)
	PutCaptures(caps)
}

func TestMuxScanReplaceUntilSlashDigit(t *testing.T) {
	const raw = "/abc/{replace::id;until-slash;digit}"
	mux := New[int]()
	require.NoError(t, mux.Register(raw, 7))

	// pure digits, fully consumed at end of input
	node, caps, converted, ok := mux.Lookup("/abc/13455")
	require.True(t, ok)
	require.Equal(t, 7, node.Value())
	require.Equal(t, "/abc/:id", converted, "replace-only: converted equals canonical even on scan backend")
	require.Equal(t, Captures{{Key: ":id", Value: "13455"}}, *caps)
	PutCaptures(caps)

	// non-digit shrinks consumed segment; remainder leaves pos < len(input) → miss
	node, caps, converted, ok = mux.Lookup("/abc/13a55")
	requireLookupMiss(t, node, caps, converted, ok)

	// trailing input without a tail expression also misses
	node, caps, converted, ok = mux.Lookup("/abc/13455/x")
	requireLookupMiss(t, node, caps, converted, ok)
}

func TestMuxScanUnnamedReplace(t *testing.T) {
	mux := New[string]()
	require.NoError(t, mux.Register("v={replace;digit}", "x"))

	node, caps, converted, ok := mux.Lookup("v=42")
	require.True(t, ok)
	require.Equal(t, "v=", converted, "unnamed replace contributes nothing to Canonical/Converted")
	require.Equal(t, Captures{{Key: "", Value: "42"}}, *caps)
	require.Equal(t, "x", node.Value())
	PutCaptures(caps)
}

func TestMuxScanUntilBlankAndRest(t *testing.T) {
	mux := New[string]()
	require.NoError(t, mux.Register("> {keep;until-blank} {replace;rest}", "msg"))

	node, caps, converted, ok := mux.Lookup("> hello world and rest")
	require.True(t, ok)
	require.Equal(t, "> hello ", converted)
	require.Equal(t, Captures{
		{Key: "", Value: "hello"},
		{Key: "", Value: "world and rest"},
	}, *caps)
	_ = node
	PutCaptures(caps)
}

func TestMuxMixedBackendPriority(t *testing.T) {
	mux := New[string]()
	// radix pattern (until-slash)
	require.NoError(t, mux.Register("/u/{replace::id;until-slash}", "radix"))
	// scan pattern with longer literal prefix
	require.NoError(t, mux.Register("/u/abc/{replace::tail;until-slash;hexdigit}", "scan"))

	// only scan matches: hexdigit
	node, caps, _, ok := mux.Lookup("/u/abc/dead")
	require.True(t, ok)
	require.Equal(t, "scan", node.Value())
	require.Equal(t, Captures{{Key: ":tail", Value: "dead"}}, *caps)
	PutCaptures(caps)

	// only radix matches: id segment is not hex
	node, caps, _, ok = mux.Lookup("/u/xyz")
	require.True(t, ok)
	require.Equal(t, "radix", node.Value())
	PutCaptures(caps)

	// only radix matches: literal prefix /u/abc but scan also requires hexdigit which "zzz" fails
	node, caps, _, ok = mux.Lookup("/u/abc")
	require.True(t, ok)
	require.Equal(t, "radix", node.Value())
	require.Equal(t, Captures{{Key: ":id", Value: "abc"}}, *caps)
	PutCaptures(caps)
}

func TestMuxScanLaterRegistrationWinsOnTie(t *testing.T) {
	mux := New[string]()
	require.NoError(t, mux.Register("v={replace;digit}", "first"))
	require.NoError(t, mux.Register("v={replace::n;digit}", "second"))

	node, caps, _, ok := mux.Lookup("v=42")
	require.True(t, ok)
	require.Equal(t, "second", node.Value(), "tie on literal prefix → later registered wins")
	PutCaptures(caps)
}

func TestMuxScanNoMatch(t *testing.T) {
	mux := New[string]()
	require.NoError(t, mux.Register("v={replace;digit}", "x"))

	node, caps, converted, ok := mux.Lookup("v=abc")
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
		rawRadix = "/p/{replace::id;until-slash}"
		rawScan  = "/p/{replace::id;until-slash;digit}"
		canonical = "/p/:id"
	)
	mux := New[int]()
	require.NoError(t, mux.Register(rawRadix, 1))
	requireMuxRegistered(t, mux, rawRadix, canonical, 1)

	// digit rule => scan backend, but canonical still /p/:id
	err := mux.Register(rawScan, 2)
	require.ErrorIs(t, err, ErrDuplicateCanonical)
	requireMuxRegistered(t, mux, rawRadix, canonical, 1)
	require.NotContains(t, mux.byRaw, rawScan)
}

func TestMuxStaticOverUntilSlashWildcard(t *testing.T) {
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

func BenchmarkMuxUntilSlashLookup(b *testing.B) {
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
