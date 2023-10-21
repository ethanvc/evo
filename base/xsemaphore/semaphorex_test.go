package xsemaphore

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewWeightedWait(t *testing.T) {
	s := NewWeighted(0)
	go func() {
		s.Go(context.Background(), func() { time.Now() })
	}()
	s.Wait()
	require.EqualValues(t, 0, s.GetSize())
	require.EqualValues(t, 0, s.GetCurrent())
	require.EqualValues(t, 0, s.GetWaitRoutines())
}
