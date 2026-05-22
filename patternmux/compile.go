package patternmux

import "strings"

type profile uint8

const (
	profileRoute profile = iota
	profileText
)

type compiledPattern struct {
	Raw             string
	Canonical       string
	CachedConverted string
	HasKeep         bool
	Profile         profile
	RoutePath       string // canonical when Profile==profileRoute; empty otherwise
	Segments        []Segment
	LiteralPrefix   int // cumulative literal chars from start (for priority tests)
}

func Compile(segments []Segment) (compiledPattern, error) {
	var b strings.Builder
	cp := compiledPattern{Segments: segments}
	for _, seg := range segments {
		switch s := seg.(type) {
		case Literal:
			cp.Raw += s.Text
			b.WriteString(s.Text)
			cp.LiteralPrefix += len(s.Text)
		case Expr:
			cp.Raw += s.Raw
			switch s.Action {
			case ActionReplace:
				if s.Wild == '*' {
					b.WriteByte('*')
					b.WriteString(s.Name)
				} else if s.Wild == ':' {
					b.WriteByte(':')
					b.WriteString(s.Name)
				}
				// unnamed replace: nothing added to canonical
			case ActionKeep:
				cp.HasKeep = true
				b.WriteString(s.Raw)
			}
		}
	}
	cp.Canonical = b.String()
	cp.Profile = assignProfile(segments)
	if !cp.HasKeep {
		cp.CachedConverted = cp.Canonical
	}
	if cp.Profile == profileRoute {
		cp.RoutePath = cp.Canonical
	}
	return cp, nil
}

func assignProfile(segments []Segment) profile {
	for _, seg := range segments {
		e, ok := seg.(Expr)
		if !ok {
			continue
		}
		if e.Action == ActionKeep {
			return profileText
		}
		for _, r := range e.Rules {
			switch r {
			case RuleUntilSlash, RuleRest:
			default:
				return profileText
			}
		}
	}
	return profileRoute
}
