package runstat

import "github.com/shirou/gopsutil/v4/mem"

type hostReader struct{}

func (r *hostReader) GetMemory() (MemoryInfo, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return MemoryInfo{}, err
	}
	return MemoryInfo{
		UsedBytes: vm.Total - vm.Available,
		MaxBytes:  vm.Total,
		Source:    SourceHost,
	}, nil
}

func newHostReader() MemoryReader {
	return &hostReader{}
}
