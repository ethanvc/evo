package runstat

import "fmt"

// CPUInfo holds CPU limit and cumulative usage for the runtime environment.
type CPUInfo struct {
	LimitCores   float64
	UsageSeconds float64
	Source       string
}

// CPUReader reads CPU limit and cumulative usage from the runtime environment.
type CPUReader interface {
	GetCPU() (CPUInfo, error)
}

// GetCPU returns CPU info from the runtime environment.
func GetCPU() (CPUInfo, error) {
	if defaultCPUReader == nil {
		return CPUInfo{}, fmt.Errorf("runstat: cpu reader not initialized")
	}
	return defaultCPUReader.GetCPU()
}

var defaultCPUReader CPUReader

func init() {
	defaultCPUReader = detectCPUReader()
}
