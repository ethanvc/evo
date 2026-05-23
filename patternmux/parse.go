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
	if len(parts) < 2 {
		return Expr{}, ErrMissingRule
	}
	head := parts[0]
	var action Action
	var name string
	switch {
	case strings.HasPrefix(head, "replace:"):
		action = ActionReplace
		name = head[len("replace:"):]
		if name == "" {
			return Expr{}, ErrInvalidSyntax
		}
	case head == "replace":
		action = ActionReplace
	case head == "keep":
		action = ActionKeep
	default:
		return Expr{}, ErrInvalidSyntax
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
