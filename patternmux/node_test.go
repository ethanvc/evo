package patternmux

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNodeAccessors(t *testing.T) {
	n := Node[int]{
		raw:             "/abc/{replace::user-id;until-slash}",
		canonical:       "/abc/:user-id",
		pattern:         "/abc/:user-id",
		hasKeep:         true,
		cachedConverted: "/abc/100",
	}

	require.Equal(t, "/abc/{replace::user-id;until-slash}", n.Raw())
	require.Equal(t, "/abc/{replace::user-id;until-slash}", n.GetPatternWithExpr())
	require.Equal(t, "/abc/:user-id", n.Canonical())
	require.Equal(t, "/abc/:user-id", n.GetPattern())
	require.True(t, n.HasKeep())
	require.Equal(t, "/abc/100", n.CachedConverted())
}

// TestNodeGetPatternEndToEnd registers each pattern shape through Mux and
// asserts the GetPattern / GetPatternWithExpr accessors on the leaf returned
// by Lookup. This proves the whole pipeline (Compile → tree.addPattern → leaf)
// preserves both forms.
func TestNodeGetPatternEndToEnd(t *testing.T) {
	cases := []struct {
		name           string
		raw            string
		input          string
		wantWithExpr   string
		wantPattern    string
	}{
		{
			name:         "named replace until-slash",
			raw:          "/abc/{replace::user-id;until-slash}",
			input:        "/abc/13455",
			wantWithExpr: "/abc/{replace::user-id;until-slash}",
			wantPattern:  "/abc/:user-id",
		},
		{
			name:         "named replace catch-all",
			raw:          "/abc/{replace:*path;rest}",
			input:        "/abc/a/b/c",
			wantWithExpr: "/abc/{replace:*path;rest}",
			wantPattern:  "/abc/*path",
		},
		{
			name:         "unnamed replace falls back to noname",
			raw:          "v={replace;digit}",
			input:        "v=42",
			wantWithExpr: "v={replace;digit}",
			wantPattern:  "v=" + PlaceholderName,
		},
		{
			name:         "keep is always unnamed → noname",
			raw:          "error code {keep;digit}",
			input:        "error code 9999",
			wantWithExpr: "error code {keep;digit}",
			wantPattern:  "error code " + PlaceholderName,
		},
		{
			name:         "mixed keep + unnamed replace each gets a slot",
			raw:          "error code {keep;digit}, transaction-id is {replace;hexdigit}",
			input:        "error code 12, transaction-id is dead",
			wantWithExpr: "error code {keep;digit}, transaction-id is {replace;hexdigit}",
			wantPattern:  "error code " + PlaceholderName + ", transaction-id is " + PlaceholderName,
		},
		{
			name:         "literal-only pattern has no expressions",
			raw:          "/api/v1/users",
			input:        "/api/v1/users",
			wantWithExpr: "/api/v1/users",
			wantPattern:  "/api/v1/users",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mux := New[int]()
			require.NoError(t, mux.Register(tc.raw, 1))

			node, caps, _, ok := mux.Lookup(tc.input)
			require.True(t, ok, "Lookup must hit for input %q", tc.input)
			require.NotNil(t, node)

			require.Equal(t, tc.wantWithExpr, node.GetPatternWithExpr())
			require.Equal(t, tc.wantWithExpr, node.Raw(),
				"Raw() must alias GetPatternWithExpr()")
			require.Equal(t, tc.wantPattern, node.GetPattern())
			PutCaptures(caps)
		})
	}
}
