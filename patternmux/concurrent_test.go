package patternmux

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLookupConcurrentSafety exercises the unified tree under concurrent
// Lookups for both a replace-only pattern (Pattern-backed Converted) and a keep
// pattern (Converted assembled per call from a pooled buffer).
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
					t.Errorf("replace lookup mismatch")
				}
				PutCaptures(caps)

				node, caps, conv, ok = mux.Lookup("error code 9999")
				if !ok || node.Value() != 2 || conv != "error code 9999" {
					t.Errorf("keep lookup mismatch: conv=%q", conv)
				}
				PutCaptures(caps)
			}
		}()
	}
	wg.Wait()
}
