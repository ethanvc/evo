package evolog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTraceId(t *testing.T) {
	id := NewTraceId()
	require.Equal(t, 32, len(id))
}
