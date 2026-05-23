package patternmux

import "strings"

// compiledPattern is the post-Compile bundle threaded into Register / addPattern.
//
// The tree is now unified (see tree.go); we no longer pre-classify patterns
// into separate backends. CachedConverted is filled when the pattern has no
// `keep` expression, so Lookup can return the constant string without
// allocating a buffer.
type compiledPattern struct {
	Raw             string
	Canonical       string
	CachedConverted string
	HasKeep         bool
	Segments        []Segment
	LiteralPrefix   int // cumulative literal chars from start (priority signal)
}

func Compile(segments []Segment) (compiledPattern, error) {
	var canonical strings.Builder
	cp := compiledPattern{Segments: segments}
	for _, seg := range segments {
		switch s := seg.(type) {
		case Literal:
			cp.Raw += s.Text
			canonical.WriteString(s.Text)
			cp.LiteralPrefix += len(s.Text)
		case Expr:
			cp.Raw += s.Raw
			switch s.Action {
			case ActionReplace:
				if s.Name != "" {
					canonical.WriteString(s.Name)
				}
			case ActionKeep:
				cp.HasKeep = true
				canonical.WriteString(s.Raw)
			}
		}
	}
	cp.Canonical = canonical.String()
	if !cp.HasKeep {
		cp.CachedConverted = cp.Canonical
	}
	return cp, nil
}
