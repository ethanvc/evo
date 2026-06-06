package runstat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCPU_usesDefault(t *testing.T) {
	info, err := GetCPU()
	require.NoError(t, err)
	assert.Equal(t, SourceHost, info.Source)
	assert.Greater(t, info.LimitCores, float64(0))
	assert.GreaterOrEqual(t, info.UsageSeconds, float64(0))
}

func TestHostCPUReader_GetCPU(t *testing.T) {
	info, err := newHostCPUReader().GetCPU()
	require.NoError(t, err)
	assert.Equal(t, SourceHost, info.Source)
	assert.Greater(t, info.LimitCores, float64(0))
	assert.GreaterOrEqual(t, info.UsageSeconds, float64(0))
}
