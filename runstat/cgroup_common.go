//go:build linux

package runstat

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const v1Unlimited = uint64(1 << 62)

type cgroupPaths struct {
	v2Path     string
	v1Memory   string
	v1CPU      string
	v1CPUAcct  string
}

func parseProcCgroup(data string) cgroupPaths {
	var paths cgroupPaths
	for line := range strings.SplitSeq(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		hierarchy, controllers, path := parts[0], parts[1], parts[2]
		if hierarchy == "0" && controllers == "" {
			paths.v2Path = path
		}
		for c := range strings.SplitSeq(controllers, ",") {
			switch c {
			case "memory":
				paths.v1Memory = path
			case "cpu":
				paths.v1CPU = path
			case "cpuacct":
				paths.v1CPUAcct = path
			}
		}
	}
	return paths
}

func cgroupDir(root, relPath string) string {
	relPath = strings.TrimPrefix(relPath, "/")
	if relPath == "" {
		return root
	}
	return filepath.Join(root, relPath)
}

func cgroupV1Dir(root, relPath string) string {
	return cgroupV1ControllerDir(root, "memory", relPath)
}

func cgroupV1ControllerDir(root, controller, relPath string) string {
	relPath = strings.TrimPrefix(relPath, "/")
	if relPath == "" {
		return filepath.Join(root, controller)
	}
	return filepath.Join(root, controller, relPath)
}

func readInt64File(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
}

func readUint64File(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(data))
	if s == "max" {
		return 0, errUnlimited
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func readStringFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

var errUnlimited = strconv.ErrSyntax

var errCPUStatNotFound = errors.New("cpu.stat usage_usec not found")

func isValidV1Limit(limit uint64) bool {
	return limit > 0 && limit < v1Unlimited
}

func isValidV2Max(max string) bool {
	return max != "" && max != "max"
}
