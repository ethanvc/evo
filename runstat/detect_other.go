//go:build !linux

package runstat

func detectMemoryReader() MemoryReader {
	return newHostReader()
}
