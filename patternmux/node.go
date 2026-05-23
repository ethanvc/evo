package patternmux

// Raw returns the registered pattern verbatim, including every `{...}`
// expression in its original form. Identical to GetPatternWithExpr; kept for
// backwards compatibility with existing call sites.
func (n *Node[T]) Raw() string {
	return n.raw
}

// GetPatternWithExpr returns the registered pattern verbatim, including every
// `{...}` expression in its original form.
func (n *Node[T]) GetPatternWithExpr() string {
	return n.raw
}

// GetPattern returns the registered pattern with every `{...}` expression
// replaced by its name. Unnamed expressions (`{replace;...}`, `{keep;...}`,
// since the `keep` action does not carry a name) are replaced by
// PlaceholderName ("noname").
//
// Examples:
//
//	"/abc/{replace::user-id;until-slash}"           -> "/abc/:user-id"
//	"/abc/{replace:*path;rest}"                     -> "/abc/*path"
//	"v={replace;digit}"                             -> "v=noname"
//	"error code {keep;digit}, tx {replace;hexdigit}" -> "error code noname, tx noname"
func (n *Node[T]) GetPattern() string {
	return n.pattern
}

// Canonical returns the deduplication key used by Mux: literal text plus
// `:name` / `*name` for replace expressions; `keep` expressions are kept
// verbatim; unnamed replace expressions contribute nothing.
func (n *Node[T]) Canonical() string {
	return n.canonical
}

// HasKeep reports whether the registered pattern contains any `keep`
// expression. When true, Lookup must assemble Converted at match time.
func (n *Node[T]) HasKeep() bool {
	return n.hasKeep
}

// CachedConverted returns the precomputed Converted output for replace-only
// patterns. When HasKeep is true, this is empty and callers must obtain
// Converted from the Lookup return value.
func (n *Node[T]) CachedConverted() string {
	return n.cachedConverted
}
