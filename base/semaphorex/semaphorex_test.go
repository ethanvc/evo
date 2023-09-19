package semaphorex

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewWeightedWait(t *testing.T) {
	s := NewWeighted(0)
	s.Wait()
	require.EqualValues(t, 0, s.GetSize())
	require.EqualValues(t, 0, s.GetCurrent())
}
