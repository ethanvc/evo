package patternmux

import "strings"

// PlaceholderName is substituted for unnamed expressions when assembling the
// "Pattern" string returned by Node.GetPattern.
const PlaceholderName = "noname"

// compiledPattern is the post-Compile bundle threaded into Register / addPattern.
//
// The string fields describe the same registered pattern from different angles
// (see design.md §4). Memory aid:
//
//   - Raw:     what you registered (verbatim, including {...})
//   - Pattern: expression labels and the replace-only Lookup output template
type compiledPattern struct {
	Raw          string // what you registered
	Pattern      string // human-readable label; unnamed expr → PlaceholderName
	HasKeep      bool
	Segments     []Segment
	LiteralChars int // total literal chars in the pattern
}

func Compile(segments []Segment) (compiledPattern, error) {
	var pattern strings.Builder
	cp := compiledPattern{Segments: segments}
	for _, seg := range segments {
		switch s := seg.(type) {
		case Literal:
			cp.Raw += s.Text
			pattern.WriteString(s.Text)
			cp.LiteralChars += len(s.Text)
		case Expr:
			cp.Raw += s.Raw
			if s.Action == ActionKeep {
				cp.HasKeep = true
			}
			if s.Name != "" {
				pattern.WriteString(s.Name)
			} else {
				pattern.WriteString(PlaceholderName)
			}
		}
	}
	cp.Pattern = pattern.String()
	return cp, nil
}
