package runstat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryInfo_Ratio(t *testing.T) {
	assert.Equal(t, 0.0, MemoryInfo{}.Ratio())
	assert.InDelta(t, 0.5, MemoryInfo{UsedBytes: 50, MaxBytes: 100}.Ratio(), 1e-9)
}

func TestGetMemory_usesDefault(t *testing.T) {
	require.NotNil(t, DefaultMemoryReader)
	info, err := GetMemory()
	require.NoError(t, err)
	assert.NotEmpty(t, info.Source)
	assert.Greater(t, info.MaxBytes, uint64(0))
}

func TestHostReader_GetMemory(t *testing.T) {
	info, err := newHostReader().GetMemory()
	require.NoError(t, err)
	assert.Equal(t, SourceHost, info.Source)
	assert.LessOrEqual(t, info.UsedBytes, info.MaxBytes)
	assert.Greater(t, info.MaxBytes, uint64(0))
}
