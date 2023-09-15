package cgroup

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"testing"
)

func TestControlGroupV1_Basic(t *testing.T) {
	cg := NewControlGroupV1()
	cpuCount, _ := cg.CpuCount()
	cpuUsage, _ := cg.CpuUsageSeconds()
	memSize, _ := cg.MemorySize()
	memUsage, _ := cg.MemoryUsageBytes()

	fmt.Printf(`cgroup v1 information:
CPU Count: %d
CPU Usage Seconds: %f
Memory Size: %s(%d)
Memory Usage Bytes: %s(%d)
`, cpuCount, cpuUsage,
		humanize.Bytes(uint64(memSize)), memSize,
		humanize.Bytes(uint64(memUsage)), memUsage,
	)
}
