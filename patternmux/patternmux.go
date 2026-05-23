package patternmux

type Mux[T any] struct {
	root        *Node[T]
	maxCaptures uint16
	byRaw       map[string]struct{}
	byCanonical map[string]T
	registerSeq int
}

func New[T any]() *Mux[T] {
	return &Mux[T]{
		root:        &Node[T]{},
		byRaw:       make(map[string]struct{}),
		byCanonical: make(map[string]T),
	}
}

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
	m.byCanonical[cp.Canonical] = value
	m.registerSeq++

	// v1: radix backend only; scan backend registers metadata for v2.
	if cp.Backend != backendRadix {
		return nil
	}

	m.root.insertIndexed(cp.IndexKey, value, patternMeta{
		raw:             cp.Raw,
		canonical:       cp.Canonical,
		hasKeep:         cp.HasKeep,
		cachedConverted: cp.CachedConverted,
		literalPrefix:   cp.LiteralPrefix,
		registerOrder:   uint64(m.registerSeq),
	})

	if n := countIndexWildcards(cp.IndexKey); n > m.maxCaptures {
		m.maxCaptures = n
	}
	return nil
}

func (m *Mux[T]) Lookup(input string) (node *Node[T], captures *Captures, converted string, ok bool) {
	node, captures, ok = m.root.matchInput(input, func() *Captures {
		return newCaptures(int(m.maxCaptures))
	})
	if !ok || node == nil {
		putCaptures(captures)
		return nil, nil, "", false
	}

	return node, captures, node.cachedConverted, true
}
