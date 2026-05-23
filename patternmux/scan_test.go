package patternmux

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConsumeByRulesIntersection(t *testing.T) {
	end, ok := consumeByRules([]Rule{RuleUntilSlash, RuleDigit}, "13455/x", 0)
	require.True(t, ok)
	require.Equal(t, 5, end)

	end, ok = consumeByRules([]Rule{RuleUntilSlash, RuleDigit}, "13a55/x", 0)
	require.True(t, ok)
	require.Equal(t, 2, end, "digit boundary wins when it lies before slash")

	end, ok = consumeByRules([]Rule{RuleHexDigit}, "deadBEEF42", 0)
	require.True(t, ok)
	require.Equal(t, 10, end)

	_, ok = consumeByRules([]Rule{RuleDigit}, "abc", 0)
	require.False(t, ok, "empty consumption must fail")
}

func TestTryScanLiteralOnly(t *testing.T) {
	segs, err := Parse("hello")
	require.NoError(t, err)
	require.True(t, tryScan(segs, "hello", nil, nil))
	require.False(t, tryScan(segs, "hell", nil, nil))
	require.False(t, tryScan(segs, "hello!", nil, nil))
}

func TestTryScanWritesConvertedOnlyForKeep(t *testing.T) {
	segs, err := Parse("a={keep;digit} b={replace;digit}")
	require.NoError(t, err)
	caps := Captures{}
	buf := []byte{}
	require.True(t, tryScan(segs, "a=12 b=34", &caps, &buf))
	require.Equal(t, "a=12 b=", string(buf))
	require.Equal(t, Captures{
		{Key: "", Value: "12"},
		{Key: "", Value: "34"},
	}, caps)
}

func TestLookupConcurrentSafety(t *testing.T) {
	mux := New[int]()
	require.NoError(t, mux.Register("/u/{replace::id;until-slash}", 1))
	require.NoError(t, mux.Register("error code {keep;digit}", 2))

	const goroutines = 32
	const iters = 500

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				node, caps, conv, ok := mux.Lookup("/u/abc")
				if !ok || node.Value() != 1 || conv != "/u/:id" {
					t.Errorf("radix lookup mismatch")
				}
				PutCaptures(caps)

				node, caps, conv, ok = mux.Lookup("error code 9999")
				if !ok || node.Value() != 2 || conv != "error code 9999" {
					t.Errorf("scan lookup mismatch: conv=%q", conv)
				}
				PutCaptures(caps)
			}
		}()
	}
	wg.Wait()
}
