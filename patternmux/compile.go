package patternmux

import "strings"

// PlaceholderName is substituted for unnamed expressions when assembling the
// "Pattern" string returned by Node.GetPattern.
const PlaceholderName = "noname"

// compiledPattern is the post-Compile bundle threaded into Register / addPattern.
//
// Four string fields describe the same registered pattern from different angles
// (see design.md §4). Memory aid:
//
//   - Raw:             what you registered (verbatim, including {...})
//   - Canonical:       which route Mux thinks this is (internal dedup key)
//   - Pattern:         human-readable route label (metrics/logs; Node.GetPattern)
//   - CachedConverted: precomputed Lookup output for replace-only patterns
//                      (internal; callers use Lookup's converted return value)
//
// CachedConverted is filled only when the pattern has no `keep` expression, so
// Lookup can return a constant string without allocating a buffer.
type compiledPattern struct {
	Raw             string // what you registered
	Canonical       string // internal dedup key; replace-only output template
	Pattern         string // human-readable label; unnamed expr → PlaceholderName
	CachedConverted string // internal; = Canonical when !HasKeep, else empty
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
