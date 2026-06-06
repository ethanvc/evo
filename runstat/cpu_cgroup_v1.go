//go:build linux

package runstat

import (
	"path/filepath"
)

type cgroupV1CPUReader struct {
	root        string
	cpuPath     string
	cpuAcctPath string
}

func (r *cgroupV1CPUReader) cpuDir() string {
	return cgroupV1ControllerDir(r.root, "cpu", r.cpuPath)
}

func (r *cgroupV1CPUReader) cpuAcctDir() string {
	path := r.cpuAcctPath
	if path == "" {
		path = r.cpuPath
	}
	return cgroupV1ControllerDir(r.root, "cpuacct", path)
}

func (r *cgroupV1CPUReader) GetCPU() (CPUInfo, error) {
	limitCores, ok, err := readV1CPULimit(r.cpuDir())
	if err != nil {
		return CPUInfo{}, err
	}
	if !ok {
		return CPUInfo{}, errUnlimited
	}
	usageNanos, err := readUint64File(filepath.Join(r.cpuAcctDir(), "cpuacct.usage"))
	if err != nil {
		return CPUInfo{}, err
	}
	return CPUInfo{
		LimitCores:   limitCores,
		UsageSeconds: float64(usageNanos) / 1e9,
		Source:       SourceCgroupV1,
	}, nil
}

func tryCgroupV1CPUReader(root, cpuPath, cpuAcctPath string) CPUReader {
	if cpuPath == "" {
		return nil
	}
	cpuDir := cgroupV1ControllerDir(root, "cpu", cpuPath)
	if _, ok, err := readV1CPULimit(cpuDir); err != nil || !ok {
		return nil
	}
	return &cgroupV1CPUReader{root: root, cpuPath: cpuPath, cpuAcctPath: cpuAcctPath}
}

func readV1CPULimit(cpuDir string) (cores float64, ok bool, err error) {
	quota, err := readInt64File(filepath.Join(cpuDir, "cpu.cfs_quota_us"))
	if err != nil {
		return 0, false, err
	}
	if quota <= 0 {
		return 0, false, nil
	}
	period, err := readInt64File(filepath.Join(cpuDir, "cpu.cfs_period_us"))
	if err != nil || period <= 0 {
		return 0, false, err
	}
	return float64(quota) / float64(period), true, nil
}
