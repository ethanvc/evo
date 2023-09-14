package cgroup

import (
	"fmt"
	"os"
)

type ControlGroupV1 struct {
}

func NewControlGroupV1() ControlGroupV1 {
	return ControlGroupV1{}
}

func (cg ControlGroupV1) CpuCount() (cnt int, err error) {
	quota, err := cg.GetCpuCfsQuotaUs()
	if err != nil {
		return
	}
	if quota == -1 {
		return 0, ErrNoLimit
	}
	period, err := cg.GetCpuCfsPeriodUs()
	if err != nil {
		return
	}
	cnt = int(quota / period)
	return
}

func (cg ControlGroupV1) CpuUsageSeconds() (x int64, err error) {
	return
}

func (cg ControlGroupV1) MemorySize() (x int64, err error) {
	return
}

func (cg ControlGroupV1) MemoryUsageBytes() (x int64, err error) {
	return
}

type CpuStatV1 struct {
	NrPeriods     int64
	NrThrottled   int64
	ThrottledTime int64
	CurrentBw     int64
	NrBurst       int64
	BurstTime     int64
}

func (cg ControlGroupV1) GetCpuStat() (stat CpuStatV1, err error) {
	p := "/sys/fs/cgroup/cpu/cpu.stat"
	content, err := os.ReadFile(p)
	if err != nil {
		return
	}
	kv := map[string]*int64{
		"nr_periods":     &stat.NrPeriods,
		"nr_throttled":   &stat.NrThrottled,
		"throttled_time": &stat.ThrottledTime,
		"current_bw":     &stat.CurrentBw,
		"nr_burst":       &stat.NrBurst,
		"burst_time":     &stat.BurstTime,
	}
	if !ParseKvContentInteger(string(content), kv) {
		return stat, fmt.Errorf("parse cpu.stat failed, content: %s", content)
	}
	return
}

func (cg ControlGroupV1) GetCpuCfsPeriodUs() (int64, error) {
	p := "/sys/fs/cgroup/cpu/cpu.cfs_period_us"
	content, err := os.ReadFile(p)
	if err != nil {
		return 0, err
	}
	return ParseSingleInteger(string(content))
}

func (cg ControlGroupV1) GetCpuCfsQuotaUs() (int64, error) {
	p := "/sys/fs/cgroup/cpu/cpu.cfs_quota_us"
	content, err := os.ReadFile(p)
	if err != nil {
		return 0, err
	}
	return ParseSingleInteger(string(content))
}
