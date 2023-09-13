package evores

type Resource struct {
	cpuCount int
}

func NewResource() *Resource {
	return &Resource{}
}

func (r *Resource) CpuCount() int {
	return r.cpuCount
}

var defaultResource = NewResource()

func DefaultResource() *Resource {
	return defaultResource
}
