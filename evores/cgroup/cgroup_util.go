package cgroup

import (
	"strconv"
	"strings"
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
		tmp, err := strconv.ParseInt(ss[i+1], 10, 64)
		if err != nil {
			return false
		}
		*num = tmp
	}
	return true
}