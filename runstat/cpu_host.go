package runstat

import (
	"runtime"

	"github.com/shirou/gopsutil/v4/cpu"
)

type hostCPUReader struct{}

func (r *hostCPUReader) GetCPU() (CPUInfo, error) {
	times, err := cpu.Times(false)
	if err != nil {
		return CPUInfo{}, err
	}
	t := times[0]
	usage := t.User + t.System + t.Nice + t.Irq + t.Softirq + t.Steal
	return CPUInfo{
		LimitCores:   float64(runtime.NumCPU()),
		UsageSeconds: usage,
		Source:       SourceHost,
	}, nil
}

func newHostCPUReader() CPUReader {
	return &hostCPUReader{}
}
