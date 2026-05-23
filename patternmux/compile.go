package patternmux

import "strings"

type profile uint8

const (
	profileRoute profile = iota
	profileText
)

// Internal route-path markers; not part of Canonical. Behavior comes from rules at compile time.
const (
	routeParamMark    = "\x00P"
	routeCatchAllMark = "\x00R"
)

type compiledPattern struct {
	Raw             string
	Canonical       string
	CachedConverted string
	HasKeep         bool
	Profile         profile
	RoutePath       string // lowered route index; empty when Profile != profileRoute
	Segments        []Segment
	LiteralPrefix   int // cumulative literal chars from start (for priority tests)
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
	cp.Profile = assignProfile(segments)
	if !cp.HasKeep {
		cp.CachedConverted = cp.Canonical
	}
	if cp.Profile == profileRoute {
		cp.RoutePath = compileRoutePath(segments)
	}
	return cp, nil
}

func compileRoutePath(segments []Segment) string {
	var b strings.Builder
	for _, seg := range segments {
		switch s := seg.(type) {
		case Literal:
			b.WriteString(s.Text)
		case Expr:
			if s.Action != ActionReplace || s.Name == "" {
				continue
			}
			if hasRule(s.Rules, RuleRest) {
				b.WriteString(routeCatchAllMark)
			} else {
				b.WriteString(routeParamMark)
			}
			b.WriteString(s.Name)
		}
	}
	return b.String()
}

func hasRule(rules []Rule, want Rule) bool {
	for _, r := range rules {
		if r == want {
			return true
		}
	}
	return false
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
