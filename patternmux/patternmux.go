package patternmux

import "sync"

// Mux is a generic pattern multiplexer. All registered patterns live in a
// single radix tree (see tree.go). Captures are pooled; the converted buffer
// for `keep` patterns is also pooled.
type Mux[T any] struct {
	root        *Node[T]
	maxCaptures uint16
	byRaw       map[string]struct{}
	byCanonical map[string]struct{}
	registerSeq int
}

func New[T any]() *Mux[T] {
	return &Mux[T]{
		root:        &Node[T]{},
		byRaw:       make(map[string]struct{}),
		byCanonical: make(map[string]struct{}),
	}
}

// Register parses, compiles, and installs `pattern` → `value` in the tree.
//
// Errors:
//   - ErrDuplicatePattern   when the same Raw was already registered.
//   - ErrDuplicateCanonical when a different Raw produced the same Canonical.
//   - parser/syntax errors from Parse.
func (m *Mux[T]) Register(pattern string, value T) error {
	segs, err := Parse(pattern)
	if err != nil {
		return err
	}

	cp, err := Compile(segs)
	if err != nil {
		return err
	}

	if _, exists := m.byRaw[cp.Raw]; exists {
		return ErrDuplicatePattern
	}
	if _, exists := m.byCanonical[cp.Canonical]; exists {
		return ErrDuplicateCanonical
	}

	m.byRaw[cp.Raw] = struct{}{}
	m.byCanonical[cp.Canonical] = struct{}{}
	m.registerSeq++

	meta := patternMeta{
		raw:             cp.Raw,
		canonical:       cp.Canonical,
		hasKeep:         cp.HasKeep,
		cachedConverted: cp.CachedConverted,
		literalPrefix:   cp.LiteralPrefix,
		registerOrder:   uint64(m.registerSeq),
	}
	m.root.addPattern(segs, value, meta)

	if n := countCaptureSites(segs); n > m.maxCaptures {
		m.maxCaptures = n
	}
	return nil
}

// Lookup walks the tree once. On a hit, captures is a pooled slice the caller
// must release with PutCaptures; converted is either the cached canonical
// string or a freshly-built string assembled from Literal + `keep` expressions.
func (m *Mux[T]) Lookup(input string) (node *Node[T], captures *Captures, converted string, ok bool) {
	node, captures, ok = m.root.matchInput(input, func() *Captures {
		return newCaptures(int(m.maxCaptures))
	})
	if !ok {
		putCaptures(captures)
		return nil, nil, "", false
	}
	if node.hasKeep {
		converted = assembleConverted(node.segments, captures)
	} else {
		converted = node.cachedConverted
	}
	return node, captures, converted, true
}

// convertedBufPool reuses byte buffers used during Converted assembly.
var convertedBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 64)
		return &b
	},
}

// assembleConverted walks the pattern's original segments and builds the
// Converted string by emitting every Literal verbatim and inlining each
// `keep` expression's captured value (replace expressions contribute nothing).
//
// Captures are appended to `caps` in segment order — one per Expr — so we walk
// them in lockstep with the segment list.
func assembleConverted(segments []Segment, caps *Captures) string {
	bp := convertedBufPool.Get().(*[]byte)
	*bp = (*bp)[:0]
	defer func() {
		*bp = (*bp)[:0]
		convertedBufPool.Put(bp)
	}()

	capIdx := 0
	for _, s := range segments {
		switch v := s.(type) {
		case Literal:
			*bp = append(*bp, v.Text...)
		case Expr:
			if v.Action == ActionKeep {
				*bp = append(*bp, (*caps)[capIdx].Value...)
			}
			capIdx++
		}
	}
	return string(*bp)
}
