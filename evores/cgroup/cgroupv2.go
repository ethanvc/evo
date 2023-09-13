package cgroup

type ControlGroupV2 struct {
}

func NewControlGroupV2() ControlGroupV2 {
	return ControlGroupV2{}
}

func (cg ControlGroupV2) CpuCount() int {
	return 0
}
