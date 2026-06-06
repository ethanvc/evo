//go:build linux

package runstat

import "os"

const procSelfCgroup = "/proc/self/cgroup"

const cgroupRoot = "/sys/fs/cgroup"

var readProcCgroup = defaultReadProcCgroup

func defaultReadProcCgroup() (string, error) {
	data, err := os.ReadFile(procSelfCgroup)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func detectMemoryReader() MemoryReader {
	return detectMemoryReaderForLinux(cgroupRoot)
}

func detectMemoryReaderForLinux(root string) MemoryReader {
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

func detectCPUReader() CPUReader {
	return detectCPUReaderForLinux(cgroupRoot)
}

func detectCPUReaderForLinux(root string) CPUReader {
	data, err := readProcCgroup()
	if err != nil {
		return newHostCPUReader()
	}
	paths := parseProcCgroup(data)
	if r := tryCgroupV2CPUReader(root, paths.v2Path); r != nil {
		return r
	}
	if r := tryCgroupV1CPUReader(root, paths.v1CPU, paths.v1CPUAcct); r != nil {
		return r
	}
	return newHostCPUReader()
}
