package patternmux

type Mux[T any] struct {
	root         *Node[T]
	maxCaptures  uint16
	byRaw        map[string]struct{}
	byCanonical  map[string]struct{}
	registerSeq  int
	scanPatterns []scanEntry[T]
}

func New[T any]() *Mux[T] {
	return &Mux[T]{
		root:        &Node[T]{},
		byRaw:       make(map[string]struct{}),
		byCanonical: make(map[string]struct{}),
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

	switch cp.Backend {
	case backendRadix:
		m.root.insertIndexed(cp.IndexKey, value, meta)
	case backendScan:
		node := &Node[T]{
			value:           value,
			raw:             meta.raw,
			canonical:       meta.canonical,
			hasKeep:         meta.hasKeep,
			cachedConverted: meta.cachedConverted,
			literalPrefix:   meta.literalPrefix,
			registerOrder:   meta.registerOrder,
			registered:      true,
		}
		m.scanPatterns = append(m.scanPatterns, scanEntry[T]{node: node, segments: segs})
	}

	if n := countCaptureSites(segs); n > m.maxCaptures {
		m.maxCaptures = n
	}
	return nil
}

func (m *Mux[T]) Lookup(input string) (node *Node[T], captures *Captures, converted string, ok bool) {
	rNode, rCaps, rOK := m.root.matchInput(input, func() *Captures {
		return newCaptures(int(m.maxCaptures))
	})

	sNode, sCaps, sConverted, sOK := m.scanLookup(input)

	switch {
	case !rOK && !sOK:
		putCaptures(rCaps)
		putCaptures(sCaps)
		return nil, nil, "", false
	case rOK && !sOK:
		return rNode, rCaps, rNode.cachedConverted, true
	case !rOK && sOK:
		return sNode, sCaps, sConverted, true
	default:
		if betterCandidate(sNode, rNode) {
			putCaptures(rCaps)
			return sNode, sCaps, sConverted, true
		}
		putCaptures(sCaps)
		return rNode, rCaps, rNode.cachedConverted, true
	}
}

// scanLookup linearly tries every scan-backend pattern and returns the best
// match per design §7 (longest literal prefix, then latest registered).
func (m *Mux[T]) scanLookup(input string) (node *Node[T], captures *Captures, converted string, ok bool) {
	for _, e := range m.scanPatterns {
		caps := newCaptures(int(m.maxCaptures))
		var buf *[]byte
		if e.node.hasKeep {
			buf = getConvertedBuf()
		}
		if !tryScan(e.segments, input, caps, buf) {
			putCaptures(caps)
			putConvertedBuf(buf)
			continue
		}

		var conv string
		if e.node.hasKeep {
			conv = string(*buf)
			putConvertedBuf(buf)
		} else {
			conv = e.node.cachedConverted
		}

		if node == nil || betterCandidate(e.node, node) {
			putCaptures(captures)
			node = e.node
			captures = caps
			converted = conv
			ok = true
			continue
		}
		putCaptures(caps)
	}
	return
}

// betterCandidate reports whether a should beat b under the §7 tie-break rule.
func betterCandidate[T any](a, b *Node[T]) bool {
	if a.literalPrefix != b.literalPrefix {
		return a.literalPrefix > b.literalPrefix
	}
	return a.registerOrder > b.registerOrder
}
