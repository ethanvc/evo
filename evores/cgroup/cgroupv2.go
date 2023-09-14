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
		return math.MaxInt64, nil
	}
	return int(maxTime / period), nil
}

func (cg ControlGroupV2) GetCpuMax() (maxTime, period int64, err error) {
	p := "/sys/fs/cgroup/cpu.max"
	content, err := os.ReadFile(p)
	if err != nil {
		return
	}
	return cg.ParseCpuMax(string(content))
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
