//go:build linux

package runstat

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestCgroupV2CPUReader_GetCPU(t *testing.T) {
	root := t.TempDir()
	podDir := filepath.Join(root, "kubepods", "pod-123")
	writeFile(t, filepath.Join(podDir, "cpu.max"), "100000 100000\n")
	writeFile(t, filepath.Join(podDir, "cpu.stat"), "usage_usec 2500000\n")

	r := &cgroupV2CPUReader{root: root, relPath: "/kubepods/pod-123"}
	info, err := r.GetCPU()
	require.NoError(t, err)
	assert.Equal(t, 1.0, info.LimitCores)
	assert.InDelta(t, 2.5, info.UsageSeconds, 1e-9)
	assert.Equal(t, SourceCgroupV2, info.Source)
}

func TestCgroupV1CPUReader_GetCPU(t *testing.T) {
	root := t.TempDir()
	cpuDir := filepath.Join(root, "cpu", "docker", "abc")
	cpuAcctDir := filepath.Join(root, "cpuacct", "docker", "abc")
	writeFile(t, filepath.Join(cpuDir, "cpu.cfs_quota_us"), "50000\n")
	writeFile(t, filepath.Join(cpuDir, "cpu.cfs_period_us"), "100000\n")
	writeFile(t, filepath.Join(cpuAcctDir, "cpuacct.usage"), "1500000000\n")

	r := &cgroupV1CPUReader{root: root, cpuPath: "/docker/abc", cpuAcctPath: "/docker/abc"}
	info, err := r.GetCPU()
	require.NoError(t, err)
	assert.InDelta(t, 0.5, info.LimitCores, 1e-9)
	assert.InDelta(t, 1.5, info.UsageSeconds, 1e-9)
	assert.Equal(t, SourceCgroupV1, info.Source)
}

func TestDetectCPUReader_v2(t *testing.T) {
	root := t.TempDir()
	podDir := filepath.Join(root, "kubepods", "pod-123")
	writeFile(t, filepath.Join(podDir, "cpu.max"), "100000 100000\n")
	writeFile(t, filepath.Join(podDir, "cpu.stat"), "usage_usec 1\n")

	oldRead := readProcCgroup
	readProcCgroup = func() (string, error) {
		return "0::/kubepods/pod-123\n", nil
	}
	t.Cleanup(func() { readProcCgroup = oldRead })

	r := detectCPUReaderForLinux(root)
	_, ok := r.(*cgroupV2CPUReader)
	assert.True(t, ok, "expected cgroupV2CPUReader")
}

func TestDetectCPUReader_host(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "cpu.max"), "max\n")

	oldRead := readProcCgroup
	readProcCgroup = func() (string, error) {
		return "0::/\n", nil
	}
	t.Cleanup(func() { readProcCgroup = oldRead })

	r := detectCPUReaderForLinux(root)
	_, ok := r.(*hostCPUReader)
	assert.True(t, ok, "expected hostCPUReader")
}

func TestCgroupV2Reader_GetMemory(t *testing.T) {
	root := t.TempDir()
	podDir := filepath.Join(root, "kubepods", "pod-123")
	writeFile(t, filepath.Join(podDir, "memory.current"), "1048576\n")
	writeFile(t, filepath.Join(podDir, "memory.max"), "2097152\n")

	r := &cgroupV2Reader{root: root, relPath: "/kubepods/pod-123"}
	info, err := r.GetMemory()
	require.NoError(t, err)
	assert.Equal(t, uint64(1048576), info.UsedBytes)
	assert.Equal(t, uint64(2097152), info.MaxBytes)
	assert.Equal(t, SourceCgroupV2, info.Source)
}

func TestCgroupV1Reader_GetMemory(t *testing.T) {
	root := t.TempDir()
	podDir := filepath.Join(root, "memory", "docker", "abc")
	writeFile(t, filepath.Join(podDir, "memory.usage_in_bytes"), "524288\n")
	writeFile(t, filepath.Join(podDir, "memory.limit_in_bytes"), "1048576\n")

	r := &cgroupV1Reader{root: root, relPath: "/docker/abc"}
	info, err := r.GetMemory()
	require.NoError(t, err)
	assert.Equal(t, uint64(524288), info.UsedBytes)
	assert.Equal(t, uint64(1048576), info.MaxBytes)
	assert.Equal(t, SourceCgroupV1, info.Source)
}

func TestDetectMemoryReader_v2(t *testing.T) {
	root := t.TempDir()
	podDir := filepath.Join(root, "kubepods", "pod-123")
	writeFile(t, filepath.Join(podDir, "memory.current"), "100\n")
	writeFile(t, filepath.Join(podDir, "memory.max"), "200\n")

	oldRead := readProcCgroup
	readProcCgroup = func() (string, error) {
		return "0::/kubepods/pod-123\n", nil
	}
	t.Cleanup(func() { readProcCgroup = oldRead })

	r := detectMemoryReaderForLinux(root)
	_, ok := r.(*cgroupV2Reader)
	assert.True(t, ok, "expected cgroupV2Reader")
}

func TestDetectMemoryReader_v1(t *testing.T) {
	root := t.TempDir()
	podDir := filepath.Join(root, "memory", "docker", "abc")
	writeFile(t, filepath.Join(podDir, "memory.usage_in_bytes"), "100\n")
	writeFile(t, filepath.Join(podDir, "memory.limit_in_bytes"), "200\n")

	oldRead := readProcCgroup
	readProcCgroup = func() (string, error) {
		return "12:memory:/docker/abc\n0::/\n", nil
	}
	t.Cleanup(func() { readProcCgroup = oldRead })

	// v2 root has unlimited max
	writeFile(t, filepath.Join(root, "memory.max"), "max\n")

	r := detectMemoryReaderForLinux(root)
	_, ok := r.(*cgroupV1Reader)
	assert.True(t, ok, "expected cgroupV1Reader")
}

func TestDetectMemoryReader_host(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "memory.max"), "max\n")
	podDir := filepath.Join(root, "memory", "docker", "abc")
	writeFile(t, filepath.Join(podDir, "memory.limit_in_bytes"), "9223372036854771712\n")

	oldRead := readProcCgroup
	readProcCgroup = func() (string, error) {
		return "12:memory:/docker/abc\n0::/\n", nil
	}
	t.Cleanup(func() { readProcCgroup = oldRead })

	r := detectMemoryReaderForLinux(root)
	_, ok := r.(*hostReader)
	assert.True(t, ok, "expected hostReader")
}

func TestParseProcCgroup(t *testing.T) {
	data := `12:memory:/docker/abc
11:cpu,cpuacct:/docker/abc
0::/kubepods/pod-123
10:cpuset:/`
	paths := parseProcCgroup(data)
	assert.Equal(t, "/kubepods/pod-123", paths.v2Path)
	assert.Equal(t, "/docker/abc", paths.v1Memory)
	assert.Equal(t, "/docker/abc", paths.v1CPU)
	assert.Equal(t, "/docker/abc", paths.v1CPUAcct)
}

func TestIsValidV1Limit(t *testing.T) {
	assert.False(t, isValidV1Limit(0))
	assert.True(t, isValidV1Limit(1024))
	assert.False(t, isValidV1Limit(v1Unlimited))
}

func TestIsValidV2Max(t *testing.T) {
	assert.False(t, isValidV2Max("max"))
	assert.True(t, isValidV2Max("1048576"))
}
