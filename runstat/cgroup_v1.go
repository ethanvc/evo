//go:build linux

package runstat

import "path/filepath"

type cgroupV1Reader struct {
	root    string
	relPath string
}

func (r *cgroupV1Reader) dir() string {
	return cgroupV1Dir(r.root, r.relPath)
}

func (r *cgroupV1Reader) GetMemory() (MemoryInfo, error) {
	dir := r.dir()
	used, err := readUint64File(filepath.Join(dir, "memory.usage_in_bytes"))
	if err != nil {
		return MemoryInfo{}, err
	}
	max, err := readUint64File(filepath.Join(dir, "memory.limit_in_bytes"))
	if err != nil {
		return MemoryInfo{}, err
	}
	if !isValidV1Limit(max) {
		return MemoryInfo{}, errUnlimited
	}
	return MemoryInfo{
		UsedBytes: used,
		MaxBytes:  max,
		Source:    SourceCgroupV1,
	}, nil
}

func tryCgroupV1Reader(root, relPath string) MemoryReader {
	if relPath == "" {
		return nil
	}
	dir := cgroupV1Dir(root, relPath)
	max, err := readUint64File(filepath.Join(dir, "memory.limit_in_bytes"))
	if err != nil || !isValidV1Limit(max) {
		return nil
	}
	return &cgroupV1Reader{root: root, relPath: relPath}
}
