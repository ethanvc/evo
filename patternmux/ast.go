package patternmux

type Action string

const (
	ActionReplace Action = "replace"
	ActionKeep    Action = "keep"
)

type Rule string

const (
	RuleUntilSlash Rule = "until-slash"
	RuleUntilBlank Rule = "until-blank"
	RuleRest       Rule = "rest"
	RuleDigit      Rule = "digit"
	RuleHexDigit   Rule = "hexdigit"
)

type Segment interface {
	segment()
}

type Literal struct {
	Text string
}

func (Literal) segment() {}

type Expr struct {
	Action Action
	Name   string // without leading : or *; empty for keep / unnamed replace
	Wild   byte   // ':' or '*'; 0 if unnamed
	Rules  []Rule
	Raw    string // original `{...}` substring
}

func (Expr) segment() {}
