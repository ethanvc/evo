//go:build linux

package runstat

import "os"

const procSelfCgroup = "/proc/self/cgroup"

var readProcCgroup = defaultReadProcCgroup

func defaultReadProcCgroup() (string, error) {
	data, err := os.ReadFile(procSelfCgroup)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func detectMemoryReader(root string) MemoryReader {
	data, err := readProcCgroup()
	if err != nil {
		return newHostReader()
	}
	paths := parseProcCgroup(data)
	if r := tryCgroupV2Reader(root, paths.v2Path); r != nil {
		return r
	}
	if r := tryCgroupV1Reader(root, paths.v1Memory); r != nil {
		return r
	}
	return newHostReader()
}
