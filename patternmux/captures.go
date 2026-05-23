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

// newCaptures returns a Captures slice from the pool with capacity at least
// defaultCapturesCap. Patterns with more than defaultCapturesCap capture sites
// trigger one slice grow on the first lookup; the pool then retains the grown
// backing array, so subsequent lookups reuse it without further growth.
func newCaptures() *Captures {
	v := capturesPool.Get()
	if v == nil {
		cs := make(Captures, 0, defaultCapturesCap)
		return &cs
	}
	cs := v.(*Captures)
	if cap(*cs) < defaultCapturesCap {
		*cs = make(Captures, 0, defaultCapturesCap)
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
