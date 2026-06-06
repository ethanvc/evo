//go:build linux

package runstat

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type cgroupV2CPUReader struct {
	root    string
	relPath string
}

func (r *cgroupV2CPUReader) dir() string {
	return cgroupDir(r.root, r.relPath)
}

func (r *cgroupV2CPUReader) GetCPU() (CPUInfo, error) {
	dir := r.dir()
	limitCores, ok, err := readV2CPULimit(filepath.Join(dir, "cpu.max"))
	if err != nil {
		return CPUInfo{}, err
	}
	if !ok {
		return CPUInfo{}, errUnlimited
	}
	usageUsec, err := readV2CPUUsageUsec(filepath.Join(dir, "cpu.stat"))
	if err != nil {
		return CPUInfo{}, err
	}
	return CPUInfo{
		LimitCores:   limitCores,
		UsageSeconds: float64(usageUsec) / 1e6,
		Source:       SourceCgroupV2,
	}, nil
}

func tryCgroupV2CPUReader(root, relPath string) CPUReader {
	if relPath == "" {
		return nil
	}
	dir := cgroupDir(root, relPath)
	if _, ok, err := readV2CPULimit(filepath.Join(dir, "cpu.max")); err != nil || !ok {
		return nil
	}
	return &cgroupV2CPUReader{root: root, relPath: relPath}
}

func readV2CPULimit(path string) (cores float64, ok bool, err error) {
	s, err := readStringFile(path)
	if err != nil {
		return 0, false, err
	}
	s = strings.TrimSpace(s)
	if s == "" || s == "max" {
		return 0, false, nil
	}
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return 0, false, nil
	}
	quota, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, false, err
	}
	period, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil || period == 0 {
		return 0, false, err
	}
	return float64(quota) / float64(period), true, nil
}

func readV2CPUUsageUsec(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 2 && fields[0] == "usage_usec" {
			return strconv.ParseUint(fields[1], 10, 64)
		}
	}
	return 0, errCPUStatNotFound
}
