package patternmux

// Tree is a unified radix index. Every registered pattern lives in this tree,
// regardless of which rules its expressions use. A wildcard segment is a node
// whose `spec` decides how it consumes input at Lookup time; multiple wildcard
// children may coexist at the same parent and are tried in priority order.
//
// Insertion never panics on static/wildcard mixing: a parent may carry both
// static `children` (indexed by first byte) and wildcard `wildcards` (linear).
// On lookup, statics are tried first; wildcards are tried in registration order
// (later registrations have higher priority per design §6 tie-break).

import "strings"

type boundaryKind uint8

const (
	boundaryNone  boundaryKind = iota // no boundary; consume to end of input (rest)
	boundarySlash                     // until-slash
	boundaryBlank                     // until-blank
)

type classKind uint8

const (
	classAny classKind = iota
	classDigit
	classHex
)

// wildcardSpec encodes a single wildcard's consume rule.
//
// `keep` does not affect routing — it only controls Converted assembly.
// `name` is the capture key written into Captures.
type wildcardSpec struct {
	boundary boundaryKind
	class    classKind
	keep     bool
	name     string
}

func (s wildcardSpec) eq(o wildcardSpec) bool {
	return s.boundary == o.boundary &&
		s.class == o.class &&
		s.keep == o.keep &&
		s.name == o.name
}

// consume returns the longest prefix of input that satisfies both the
// boundary and class constraints (§2.1: rules apply simultaneously to the
// same span). Returns 0 if no character may be consumed.
func (s wildcardSpec) consume(input string) int {
	end := 0
	for end < len(input) {
		c := input[end]
		switch s.boundary {
		case boundarySlash:
			if c == '/' {
				return end
			}
		case boundaryBlank:
			if isBlank(c) {
				return end
			}
		}
		switch s.class {
		case classDigit:
			if !isDigit(c) {
				return end
			}
		case classHex:
			if !isHexDigit(c) {
				return end
			}
		}
		end++
	}
	return end
}

func specFromExpr(e Expr) wildcardSpec {
	s := wildcardSpec{
		keep: e.Action == ActionKeep,
		name: e.Name,
	}
	for _, r := range e.Rules {
		switch r {
		case RuleUntilSlash:
			s.boundary = boundarySlash
		case RuleUntilBlank:
			s.boundary = boundaryBlank
		case RuleRest:
			s.boundary = boundaryNone
		case RuleDigit:
			s.class = classDigit
		case RuleHexDigit:
			s.class = classHex
		}
	}
	return s
}

func isBlank(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') ||
		(b >= 'a' && b <= 'f') ||
		(b >= 'A' && b <= 'F')
}

// patternMeta is registration-time metadata copied onto a leaf node.
type patternMeta struct {
	raw           string
	pattern       string
	hasKeep       bool
	literalChars  int
	registerOrder uint64
}

type Node[T any] struct {
	// Leaf metadata (only meaningful when registered).
	value         T
	raw           string
	pattern       string
	hasKeep       bool
	literalChars  int
	registerOrder uint64
	registered    bool
	segments      []Segment // populated only when hasKeep, for Converted assembly

	// Static branch fields. prefix is the literal text on this node;
	// it is empty on the root and on wildcard nodes.
	prefix   string
	indices  string
	children []*Node[T]

	// Wildcard branch fields.
	spec   wildcardSpec
	isWild bool

	// Wildcard children of this node. Tried after `children` in matchInput.
	wildcards []*Node[T]
}

// addPattern installs `value` + `meta` as the leaf for the given segment list.
// Caller has already deduplicated by raw.
func (n *Node[T]) addPattern(segments []Segment, value T, meta patternMeta) {
	cur := n
	for _, s := range segments {
		switch v := s.(type) {
		case Literal:
			cur = cur.descendOrSplitLiteral(v.Text)
		case Expr:
			cur = cur.descendOrAddWildcard(specFromExpr(v))
		}
	}
	cur.value = value
	cur.registered = true
	cur.raw = meta.raw
	cur.pattern = meta.pattern
	cur.hasKeep = meta.hasKeep
	cur.literalChars = meta.literalChars
	cur.registerOrder = meta.registerOrder
	if meta.hasKeep {
		cur.segments = segments
	}
}

// descendOrSplitLiteral walks `text` into the static-prefix subtree, splitting
// nodes as needed. Returns the node where insertion has reached `text`'s end.
func (n *Node[T]) descendOrSplitLiteral(text string) *Node[T] {
	cur := n
	for {
		// On a wildcard or a fresh root node with no own prefix yet,
		// the text simply attaches as the prefix or descends into a child.
		if cur.isWild || (cur.prefix == "" && len(cur.children) == 0 && !cur.registered && len(cur.wildcards) == 0) {
			// Empty static-only node: own the prefix.
			if !cur.isWild && cur.prefix == "" && len(cur.children) == 0 && !cur.registered && len(cur.wildcards) == 0 {
				cur.prefix = text
				return cur
			}
			// Wildcard node: descend into a static child for `text`.
			child := cur.findOrCreateStaticChild(text[0])
			if child.prefix == "" {
				child.prefix = text
				return child
			}
			cur = child
			continue
		}

		i := commonPrefixLen(cur.prefix, text)
		if i < len(cur.prefix) {
			// Split: extract the diverging tail of cur.prefix into a new child,
			// preserving cur's existing leaf metadata and children on the split.
			tail := &Node[T]{
				prefix:        cur.prefix[i:],
				indices:       cur.indices,
				children:      cur.children,
				wildcards:     cur.wildcards,
				value:         cur.value,
				registered:    cur.registered,
				raw:           cur.raw,
				pattern:       cur.pattern,
				hasKeep:       cur.hasKeep,
				literalChars:  cur.literalChars,
				registerOrder: cur.registerOrder,
				segments:      cur.segments,
			}
			cur.prefix = cur.prefix[:i]
			cur.indices = string([]byte{tail.prefix[0]})
			cur.children = []*Node[T]{tail}
			cur.wildcards = nil
			cur.value = *new(T)
			cur.registered = false
			cur.raw = ""
			cur.pattern = ""
			cur.hasKeep = false
			cur.literalChars = 0
			cur.registerOrder = 0
			cur.segments = nil
		}

		if i == len(text) {
			// `text` exactly matches cur.prefix (after possible split).
			return cur
		}

		// Some text remains; descend into matching child or create new.
		text = text[i:]
		child := cur.findOrCreateStaticChild(text[0])
		if child.prefix == "" {
			child.prefix = text
			return child
		}
		cur = child
	}
}

// findOrCreateStaticChild returns the static child whose prefix begins with c,
// creating an empty placeholder if necessary.
func (n *Node[T]) findOrCreateStaticChild(c byte) *Node[T] {
	for i := 0; i < len(n.indices); i++ {
		if n.indices[i] == c {
			return n.children[i]
		}
	}
	n.indices += string([]byte{c})
	child := &Node[T]{}
	n.children = append(n.children, child)
	return child
}

// descendOrAddWildcard returns the existing wildcard child whose spec matches,
// or installs a new one. New wildcards are prepended so later registrations
// are tried first, implementing the design §6 tie-break (latest registration
// wins on equal literal prefix).
func (n *Node[T]) descendOrAddWildcard(spec wildcardSpec) *Node[T] {
	for _, wc := range n.wildcards {
		if wc.spec.eq(spec) {
			return wc
		}
	}
	wc := &Node[T]{
		isWild: true,
		spec:   spec,
	}
	n.wildcards = append([]*Node[T]{wc}, n.wildcards...)
	return wc
}

func commonPrefixLen(a, b string) int {
	max := len(a)
	if len(b) < max {
		max = len(b)
	}
	i := 0
	for i < max && a[i] == b[i] {
		i++
	}
	return i
}

// matchInput walks the tree against input. Returns the leaf node, captures, ok.
//
// Strategy at each node:
//  1. Strip the node's static prefix; if input is shorter than prefix, miss.
//  2. If input has more bytes, try to descend into the matching static child.
//     Static success short-circuits (longest literal prefix wins, design §6).
//  3. If statics miss, walk wildcards in registration order; the first whose
//     consume + downstream lookup succeeds wins. Captures written by a failed
//     wildcard branch are truncated before trying the next.
func (n *Node[T]) matchInput(input string) (*Node[T], *Captures, bool) {
	return n.matchAt(input, nil)
}

func (n *Node[T]) matchAt(input string, caps *Captures) (*Node[T], *Captures, bool) {
	if len(input) < len(n.prefix) || !strings.HasPrefix(input, n.prefix) {
		return nil, caps, false
	}
	input = input[len(n.prefix):]

	if len(input) == 0 {
		if n.registered {
			return n, caps, true
		}
		// Fall through to wildcards: any wildcard whose consume returns 0 is
		// skipped, so a node without a leaf and no zero-consuming wildcard
		// will simply miss.
	} else {
		idxc := input[0]
		for i := 0; i < len(n.indices); i++ {
			if n.indices[i] == idxc {
				savedLen := capsLen(caps)
				leaf, c, ok := n.children[i].matchAt(input, caps)
				if ok {
					return leaf, c, true
				}
				caps = c
				truncCaps(caps, savedLen)
				// Only one static child may begin with idxc; fall through to
				// wildcards instead of trying further siblings.
				break
			}
		}
	}

	for _, wc := range n.wildcards {
		end := wc.spec.consume(input)
		if end == 0 {
			continue
		}
		savedLen := capsLen(caps)
		if caps == nil {
			caps = newCaptures()
		}
		i := len(*caps)
		*caps = (*caps)[:i+1]
		(*caps)[i] = Capture{Key: wc.spec.name, Value: input[:end]}

		rem := input[end:]
		if len(rem) == 0 {
			if wc.registered {
				return wc, caps, true
			}
		} else {
			leaf, c, ok := wc.matchAt(rem, caps)
			if ok {
				return leaf, c, true
			}
			caps = c
		}
		truncCaps(caps, savedLen)
	}

	return nil, caps, false
}

func capsLen(caps *Captures) int {
	if caps == nil {
		return 0
	}
	return len(*caps)
}

func truncCaps(caps *Captures, n int) {
	if caps != nil {
		*caps = (*caps)[:n]
	}
}
