package patternmux

import "strings"

// matchBackend selects the Lookup implementation from expression rules.
type matchBackend uint8

const (
	backendRadix matchBackend = iota // until-slash + rest rules → radix index
	backendScan                      // keep, digit, hexdigit, … → linear scan (v2)
)

// Lowered index markers; not part of Canonical. Semantics come from rules at compile time.
const (
	markUntilSlash = "\x00S" // until-slash rule
	markRest       = "\x00R" // rest rule
)

// untilSlashBoundary is the consume boundary for the until-slash rule.
const untilSlashBoundary = '/'

type compiledPattern struct {
	Raw             string
	Canonical       string
	CachedConverted string
	HasKeep         bool
	Backend         matchBackend
	IndexKey        string // lowered pattern for radix backend; empty otherwise
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
	cp.Backend = selectBackend(segments)
	if !cp.HasKeep {
		cp.CachedConverted = cp.Canonical
	}
	if cp.Backend == backendRadix {
		cp.IndexKey = compileIndexKey(segments)
	}
	return cp, nil
}

func compileIndexKey(segments []Segment) string {
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
				b.WriteString(markRest)
			} else {
				b.WriteString(markUntilSlash)
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

func selectBackend(segments []Segment) matchBackend {
	for _, seg := range segments {
		e, ok := seg.(Expr)
		if !ok {
			continue
		}
		if e.Action == ActionKeep {
			return backendScan
		}
		// Unnamed replace contributes no key to the radix index; route via scan.
		if e.Action == ActionReplace && e.Name == "" {
			return backendScan
		}
		for _, r := range e.Rules {
			switch r {
			case RuleUntilSlash, RuleRest:
			default:
				return backendScan
			}
		}
	}
	return backendRadix
}

// countCaptureSites returns the number of Exprs in segments. Both backends emit
// one Capture per Expr (named, unnamed, keep — all participate per design §5).
func countCaptureSites(segments []Segment) uint16 {
	var n uint16
	for _, s := range segments {
		if _, ok := s.(Expr); ok {
			n++
		}
	}
	return n
}
