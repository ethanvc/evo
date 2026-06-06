package runstat

import "path/filepath"

type cgroupV2Reader struct {
	root    string
	relPath string
}

func (r *cgroupV2Reader) dir() string {
	return cgroupDir(r.root, r.relPath)
}

func (r *cgroupV2Reader) GetMemory() (MemoryInfo, error) {
	dir := r.dir()
	used, err := readUint64File(filepath.Join(dir, "memory.current"))
	if err != nil {
		return MemoryInfo{}, err
	}
	maxStr, err := readStringFile(filepath.Join(dir, "memory.max"))
	if err != nil {
		return MemoryInfo{}, err
	}
	if !isValidV2Max(maxStr) {
		return MemoryInfo{}, errUnlimited
	}
	max, err := readUint64File(filepath.Join(dir, "memory.max"))
	if err != nil {
		return MemoryInfo{}, err
	}
	return MemoryInfo{
		UsedBytes: used,
		MaxBytes:  max,
		Source:    SourceCgroupV2,
	}, nil
}

func tryCgroupV2Reader(root, relPath string) MemoryReader {
	if relPath == "" {
		return nil
	}
	dir := cgroupDir(root, relPath)
	maxStr, err := readStringFile(filepath.Join(dir, "memory.max"))
	if err != nil || !isValidV2Max(maxStr) {
		return nil
	}
	return &cgroupV2Reader{root: root, relPath: relPath}
}
