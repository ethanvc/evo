package patternmux

import "strings"

// PlaceholderName is substituted for unnamed expressions when assembling the
// "Pattern" string returned by Node.GetPattern.
const PlaceholderName = "noname"

// compiledPattern is the post-Compile bundle threaded into Register / addPattern.
//
// The tree is now unified (see tree.go); we no longer pre-classify patterns
// into separate backends. CachedConverted is filled when the pattern has no
// `keep` expression, so Lookup can return the constant string without
// allocating a buffer.
type compiledPattern struct {
	Raw             string
	Canonical       string
	Pattern         string // every Expr replaced by its Name (or PlaceholderName)
	CachedConverted string
	HasKeep         bool
	Segments        []Segment
	LiteralPrefix   int // cumulative literal chars from start (priority signal)
}

func Compile(segments []Segment) (compiledPattern, error) {
	var canonical, pattern strings.Builder
	cp := compiledPattern{Segments: segments}
	for _, seg := range segments {
		switch s := seg.(type) {
		case Literal:
			cp.Raw += s.Text
			canonical.WriteString(s.Text)
			pattern.WriteString(s.Text)
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
			if s.Name != "" {
				pattern.WriteString(s.Name)
			} else {
				pattern.WriteString(PlaceholderName)
			}
		}
	}
	cp.Canonical = canonical.String()
	cp.Pattern = pattern.String()
	if !cp.HasKeep {
		cp.CachedConverted = cp.Canonical
	}
	return cp, nil
}
