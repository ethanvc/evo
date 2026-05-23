package patternmux

// Value returns the value installed at this leaf during Register. Reading
// Value on a non-leaf node (a tree internal) returns the zero value of T.
func (n *Node[T]) Value() T { return n.value }

// GetPatternWithExpr returns the registered pattern verbatim, including every
// `{...}` expression in its original form. Use this for debugging or for
// faithfully echoing what was registered; for human-readable route IDs prefer
// GetPattern.
func (n *Node[T]) GetPatternWithExpr() string {
	return n.raw
}

// GetPattern returns the registered pattern with every `{...}` expression
// replaced by its name. Unnamed expressions (`{replace;...}`, `{keep;...}`)
// are replaced by PlaceholderName ("noname"). Suitable for use as a
// human-readable route identifier in metrics, logs, traces.
//
// Examples:
//
//	"/abc/{replace::user-id;until-slash}"            -> "/abc/:user-id"
//	"/abc/{replace:*path;rest}"                      -> "/abc/*path"
//	"v={replace;digit}"                              -> "v=noname"
//	"error code {keep;digit}, tx {replace;hexdigit}" -> "error code noname, tx noname"
func (n *Node[T]) GetPattern() string {
	return n.pattern
}

// HasKeep reports whether the registered pattern contains any `keep`
// expression. When true, Lookup assembles its `converted` return value at
// match time; otherwise `converted` is a constant precomputed at Register.
// Most callers do not need this; it exists for callers that want to know
// whether a route's converted output is dynamic.
func (n *Node[T]) HasKeep() bool {
	return n.hasKeep
}
