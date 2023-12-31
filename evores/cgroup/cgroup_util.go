package cgroup

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

var (
	ErrNoLimit = errors.New("NoLimit")
)

func ParseKvContentInteger(content string, kv map[string]*int64) bool {
	// content like:
	// k1 v1
	// k2 v2
	// ....
	ss := strings.Fields(content)
	if (len(ss) % 2) == 1 {
		return false
	}
	for i := 0; i < len(ss); i += 2 {
		num := kv[ss[i]]
		if num == nil {
			continue
		}
		tmp, err := strconv.ParseInt(ss[i+1], 10, 64)
		if err != nil {
			return false
		}
		*num = tmp
	}
	return true
}

func ParseSingleInteger(content string) (int64, error) {
	content = strings.TrimSpace(content)
	if content == "max" {
		return math.MaxInt64, nil
	}
	num, err := strconv.ParseInt(content, 10, 64)
	if err != nil {
		return 0, err
	}
	return num, nil
}
