package cgroup

import (
	"fmt"
	"testing"
)

func TestControlGroupV2_Basic(t *testing.T) {
	cg := NewControlGroupV2()
	cnt, err := cg.CpuCount()
	fmt.Println(cnt, err)
}
