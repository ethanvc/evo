//go:build !linux

package runstat

func detectMemoryReader(root string) MemoryReader {
	return newHostReader()
}
