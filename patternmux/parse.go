package patternmux

import (
	"strings"
)

var knownRules = map[Rule]struct{}{
	RuleUntilSlash: {},
	RuleUntilBlank: {},
	RuleRest:       {},
	RuleDigit:      {},
	RuleHexDigit:   {},
}

func Parse(pattern string) ([]Segment, error) {
	if pattern == "" {
		return nil, ErrEmptyPattern
	}
	var segs []Segment
	var lit strings.Builder
	flushLiteral := func() {
		if lit.Len() > 0 {
			segs = append(segs, Literal{Text: lit.String()})
			lit.Reset()
		}
	}
	for i := 0; i < len(pattern); i++ {
		if pattern[i] != '{' {
			lit.WriteByte(pattern[i])
			continue
		}
		j := strings.IndexByte(pattern[i:], '}')
		if j < 0 {
			return nil, ErrInvalidSyntax
		}
		j += i
		raw := pattern[i : j+1]
		expr, err := parseExpr(raw)
		if err != nil {
			return nil, err
		}
		flushLiteral()
		segs = append(segs, expr)
		i = j
	}
	flushLiteral()
	return segs, nil
}

func parseExpr(raw string) (Expr, error) {
	inner := raw[1 : len(raw)-1]
	parts := strings.Split(inner, ";")

	// §3.1 rule 3: segment₁ must be a valid `action[:name]`. Split on the
	// first `:` to separate the action keyword from the optional name; any
	// further `:` (e.g. `replace::id`, `keep::name`) stays in the name part.
	// Validate the action shape before checking rule presence so a fully
	// empty `{}` is reported as missing-action (ErrInvalidSyntax) rather
	// than missing-rule.
	actionPart, name, hasColon := strings.Cut(parts[0], ":")
	var action Action
	switch actionPart {
	case "replace":
		action = ActionReplace
	case "keep":
		action = ActionKeep
	default:
		return Expr{}, ErrInvalidSyntax
	}
	if hasColon && name == "" {
		// `action:` with empty name (e.g. `{replace:;digit}`) is rejected:
		// writing the colon implies an intent to provide a name, so an
		// empty one is treated as a typo rather than equivalent to bare
		// `action`.
		return Expr{}, ErrInvalidSyntax
	}

	// §3.1 rule 4: at least one rule segment is required.
	if len(parts) < 2 {
		return Expr{}, ErrMissingRule
	}

	rules := make([]Rule, 0, len(parts)-1)
	for _, p := range parts[1:] {
		if p == "" {
			return Expr{}, ErrInvalidSyntax
		}
		r := Rule(p)
		if _, ok := knownRules[r]; !ok {
			return Expr{}, ErrUnknownRule
		}
		rules = append(rules, r)
	}
	return Expr{
		Action: action,
		Name:   name,
		Rules:  rules,
		Raw:    raw,
	}, nil
}
