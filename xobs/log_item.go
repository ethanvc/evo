package xobs

import (
	"slices"
	"time"
)

type Level int

const (
	LevelNotSet Level = 0
	LevelDbg    Level = 10
	LevelInfo   Level = 20
	LevelWarn   Level = 30
	LevelErr    Level = 40
)

func (l Level) String() string {
	switch l {
	case LevelNotSet:
		return "not_set"
	case LevelDbg:
		return "dbg"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelErr:
		return "err"
	}
	return "unknown"
}

type LogItem struct {
	Msg      string
	Time     time.Time
	Level    Level
	Position string
	front    [nAttrsInline]Attr
	nFront   int
	back     []Attr
	ObsCtx   *ObsContext
}

const nAttrsInline = 5

// NumAttrs returns the number of attributes in the LogItem.
func (r *LogItem) NumAttrs() int {
	return r.nFront + len(r.back)
}

// Attrs calls f on each Attr in the LogItem.
// Iteration stops if f returns false.
func (r *LogItem) Attrs(f func(Attr) bool) {
	for i := 0; i < r.nFront; i++ {
		if !f(r.front[i]) {
			return
		}
	}
	for _, a := range r.back {
		if !f(a) {
			return
		}
	}
}

// AddAttrs appends the given Attrs to the LogItem's list of Attrs.
func (r *LogItem) AddAttrs(attrs ...Attr) {
	var i int
	for i = 0; i < len(attrs) && r.nFront < len(r.front); i++ {
		r.front[r.nFront] = attrs[i]
		r.nFront++
	}
	r.back = append(r.back, attrs[i:]...)
}

// Add converts alternating key-value args to Attrs, then appends them.
// If a non-string key is encountered, it is stored under "!BADKEY".
func (r *LogItem) Add(args ...any) {
	var a Attr
	for len(args) > 0 {
		a, args = argsToAttr(args)
		if r.nFront < len(r.front) {
			r.front[r.nFront] = a
			r.nFront++
		} else {
			if r.back == nil {
				r.back = make([]Attr, 0, countArgs(args)+1)
			}
			r.back = append(r.back, a)
		}
	}
}

// Clone returns a copy of the LogItem with no shared state.
func (r LogItem) Clone() LogItem {
	r.back = slices.Clip(r.back)
	return r
}

const badKey = "!BADKEY"

func argsToAttr(args []any) (Attr, []any) {
	switch x := args[0].(type) {
	case string:
		if len(args) == 1 {
			return String(badKey, x), nil
		}
		return Any(x, args[1]), args[2:]
	case Attr:
		return x, args[1:]
	default:
		return Any(badKey, x), args[1:]
	}
}

func countArgs(args []any) int {
	n := 0
	for i := 0; i < len(args); i++ {
		n++
		if _, ok := args[i].(string); ok {
			i++
		}
	}
	return n
}
