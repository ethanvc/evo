//go:build !linux

package runstat

func detectMemoryReader() MemoryReader {
	return newHostReader()
}

func detectCPUReader() CPUReader {
	return newHostCPUReader()
}
