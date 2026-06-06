package runstat

import "fmt"

const (
	SourceCgroupV2 = "cgroup.v2"
	SourceCgroupV1 = "cgroup.v1"
	SourceHost     = "host"
)

// MemoryInfo holds current memory usage and the effective limit for the runtime
// environment (cgroup limit in containers, host MemTotal otherwise).
type MemoryInfo struct {
	UsedBytes uint64
	MaxBytes  uint64
	Source    string
}

// Ratio returns UsedBytes/MaxBytes. MaxBytes==0 yields 0.
func (m MemoryInfo) Ratio() float64 {
	if m.MaxBytes == 0 {
		return 0
	}
	return float64(m.UsedBytes) / float64(m.MaxBytes)
}

// MemoryReader reads memory usage from the runtime environment.
type MemoryReader interface {
	GetMemory() (MemoryInfo, error)
}

// GetMemory returns memory info from the runtime environment.
func GetMemory() (MemoryInfo, error) {
	if defaultMemoryReader == nil {
		return MemoryInfo{}, fmt.Errorf("runstat: memory reader not initialized")
	}
	return defaultMemoryReader.GetMemory()
}

var defaultMemoryReader MemoryReader

func init() {
	defaultMemoryReader = detectMemoryReader()
}
