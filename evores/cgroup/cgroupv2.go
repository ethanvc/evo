package cgroup

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

type ControlGroupV2 struct {
}

func NewControlGroupV2() ControlGroupV2 {
	return ControlGroupV2{}
}

func (cg ControlGroupV2) CpuCount() (cnt int, err error) {
	maxTime, period, err := cg.GetCpuMax()
	if err != nil {
		return
	}
	if maxTime == math.MaxInt64 {
		return 0, ErrNoLimit
	}
	return int(maxTime / period), nil
}

func (cg ControlGroupV2) CpuUsageSeconds() (x float64, err error) {
	return
}

func (cg ControlGroupV2) MemorySize() (cnt int64, err error) {
	return
}

func (cg ControlGroupV2) MemoryUsageBytes() (x int64, err error) {
	return
}

func (cg ControlGroupV2) GetCpuMax() (maxTime, period int64, err error) {
	p := "/sys/fs/cgroup/cpu.max"
	content, err := os.ReadFile(p)
	if err != nil {
		return
	}
	return cg.ParseCpuMax(string(content))
}

func (cg ControlGroupV2) GetCpuStat() (stat CpuStat, err error) {
	p := "/sys/fs/cgroup/cpu.stat"
	content, err := os.ReadFile(p)
	if err != nil {
		return
	}
	return cg.ParseCpuStat(string(content))
}

func (cg ControlGroupV2) ParseCpuStat(content string) (stat CpuStat, err error) {
	kv := map[string]*int64{
		"usage_usec":     &stat.UsageUsec,
		"user_usec":      &stat.UserUsec,
		"system_usec":    &stat.SystemUsec,
		"nr_periods":     &stat.NrPeriods,
		"nr_throttled":   &stat.NrThrottled,
		"throttled_usec": &stat.ThrottledUsec,
	}
	if !ParseKvContentInteger(content, kv) {
		return stat, fmt.Errorf("parse cpu.stat failed, content: %s", content)
	}
	return
}

type CpuStat struct {
	UsageUsec     int64
	UserUsec      int64
	SystemUsec    int64
	NrPeriods     int64
	NrThrottled   int64
	ThrottledUsec int64
}

func (cg ControlGroupV2) ParseCpuMax(content string) (maxTime, period int64, err error) {
	// content: "max 100000", "100000 100000"
	ss := strings.Fields(content)
	if len(ss) != 2 {
		return 0, 0, fmt.Errorf("parse cpu.max failed, raw content: %s", content)
	}
	if ss[0] == "max" {
		maxTime = math.MaxInt64
	} else {
		maxTime, err = strconv.ParseInt(ss[0], 10, 64)
		if err != nil {
			return
		}
	}
	period, err = strconv.ParseInt(ss[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse cpu.max failed, raw content: %s", content)
	}
	return
}

func (cg ControlGroupV2) GetMemoryCurrent() (int64, error) {
	p := "/sys/fs/cgroup/memory.current"
	content, err := os.ReadFile(p)
	if err != nil {
		return 0, err
	}
	return ParseSingleInteger(string(content))
}

func (cg ControlGroupV2) GetMemoryMax() (int64, error) {
	p := "/sys/fs/cgroup/memory.max"
	content, err := os.ReadFile(p)
	if err != nil {
		return 0, err
	}
	return ParseSingleInteger(string(content))
}
