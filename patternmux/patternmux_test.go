package patternmux

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMuxRegisterLookupParamRoute(t *testing.T) {
	mux := New[string]()
	require.NoError(t, mux.Register("/abc/{replace::user-id;until-slash}", "user-get"))

	node, caps, converted, ok := mux.Lookup("/abc/13455")
	require.True(t, ok)
	require.NotNil(t, node)
	require.Equal(t, "user-get", node.Value())
	require.Equal(t, "/abc/:user-id", node.Canonical())
	require.Equal(t, "/abc/:user-id", converted)
	require.Equal(t, node.CachedConverted(), converted)
	require.NotNil(t, caps)
	require.Equal(t, Captures{{Key: "user-id", Value: "13455"}}, *caps)
	PutCaptures(caps)
}

func TestMuxRegisterLookupCatchAll(t *testing.T) {
	mux := New[string]()
	require.NoError(t, mux.Register("/abc/{replace:*path;rest}", "files"))

	node, caps, converted, ok := mux.Lookup("/abc/a/b/c")
	require.True(t, ok)
	require.NotNil(t, node)
	require.Equal(t, "files", node.Value())
	require.Equal(t, "/abc/*path", converted)
	require.NotNil(t, caps)
	require.Equal(t, Captures{{Key: "path", Value: "a/b/c"}}, *caps)
	PutCaptures(caps)
}

func TestMuxLookupMiss(t *testing.T) {
	mux := New[string]()
	require.NoError(t, mux.Register("/abc/{replace::user-id;until-slash}", "v"))

	node, caps, converted, ok := mux.Lookup("/other/1")
	require.False(t, ok)
	require.Nil(t, node)
	require.Nil(t, caps)
	require.Empty(t, converted)
}

func TestMuxTextProfileRegisterOnly(t *testing.T) {
	mux := New[string]()
	require.NoError(t, mux.Register("error code {keep;digit}", "log-parser"))

	node, caps, converted, ok := mux.Lookup("error code 123456")
	require.False(t, ok)
	require.Nil(t, node)
	require.Nil(t, caps)
	require.Empty(t, converted)
}

func TestMuxDuplicateRaw(t *testing.T) {
	mux := New[int]()
	require.NoError(t, mux.Register("/a/{replace::id;until-slash}", 1))
	err := mux.Register("/a/{replace::id;until-slash}", 2)
	require.ErrorIs(t, err, ErrDuplicatePattern)
}

func TestMuxDuplicateCanonical(t *testing.T) {
	mux := New[int]()
	require.NoError(t, mux.Register("/p/{replace::id;until-slash}", 1))
	// v1 route syntax has no second raw form for the same :name canonical without
	// changing rules; adding digit switches profile but canonical stays /p/:id.
	err := mux.Register("/p/{replace::id;until-slash;digit}", 2)
	require.ErrorIs(t, err, ErrDuplicateCanonical)
}

func TestMuxStaticOverParam(t *testing.T) {
	mux := New[string]()
	require.NoError(t, mux.Register("/api/{replace::id;until-slash}", "by-id"))
	require.NoError(t, mux.Register("/api/v1/users", "static-users"))

	node, caps, _, ok := mux.Lookup("/api/v1/users")
	require.True(t, ok)
	require.Equal(t, "static-users", node.Value())
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
