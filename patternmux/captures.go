package patternmux

import "sync"

type Capture struct {
	Key   string
	Value string
}

type Captures []Capture

func (cs Captures) ByName(name string) string {
	for _, c := range cs {
		if c.Key == name {
			return c.Value
		}
	}
	return ""
}

const defaultCapturesCap = 16

var capturesPool sync.Pool

func newCaptures(wantCap int) *Captures {
	v := capturesPool.Get()
	var cs *Captures
	if v == nil {
		cs = &Captures{}
	} else {
		cs = v.(*Captures)
	}
	if wantCap < defaultCapturesCap {
		wantCap = defaultCapturesCap
	}
	if cap(*cs) < wantCap {
		*cs = make(Captures, 0, wantCap)
	} else {
		*cs = (*cs)[:0]
	}
	return cs
}

func putCaptures(cs *Captures) {
	if cs != nil {
		*cs = (*cs)[:0]
		capturesPool.Put(cs)
	}
}

// PutCaptures returns cs to the global captures pool. cs must be a non-nil
// pointer previously returned by Lookup and must not be used after PutCaptures.
func PutCaptures(cs *Captures) { putCaptures(cs) }
