package cgroup

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"testing"
)

func TestControlGroupV1_Basic(t *testing.T) {
	cg := NewControlGroupV1()
	cpuCount, err := cg.CpuCount()
	if err != nil {
		fmt.Println(err)
	}
	cpuUsage, err := cg.CpuUsageSeconds()
	if err != nil {
		fmt.Println(err)
	}
	memSize, err := cg.MemorySize()
	if err != nil {
		fmt.Println(err)
	}
	memUsage, err := cg.MemoryUsageBytes()
	if err != nil {
		fmt.Println(err)
	}

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
