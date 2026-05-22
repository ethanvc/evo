package patternmux

import "strings"

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

	// v1 supports route-profile lookup only; text profile still registers metadata.
	if cp.Profile != profileRoute {
		return nil
	}

	m.root.addRoute(cp.RoutePath, value, routeMeta{
		raw:             cp.Raw,
		canonical:       cp.Canonical,
		hasKeep:         cp.HasKeep,
		cachedConverted: cp.CachedConverted,
		literalPrefix:   cp.LiteralPrefix,
		registerOrder:   uint64(m.registerSeq),
	})

	if n := countParams(cp.RoutePath); n > m.maxCaptures {
		m.maxCaptures = n
	}
	return nil
}

func (m *Mux[T]) Lookup(input string) (node *Node[T], captures *Captures, converted string, ok bool) {
	node, captures, ok = m.root.getValue(input, func() *Captures {
		return newCaptures(int(m.maxCaptures))
	})
	if !ok || node == nil {
		putCaptures(captures)
		return nil, nil, "", false
	}

	normalizeCatchAllCapture(node.canonical, captures)
	return node, captures, node.cachedConverted, true
}

func normalizeCatchAllCapture(canonical string, captures *Captures) {
	if captures == nil {
		return
	}
	for i := range *captures {
		c := &(*captures)[i]
		if c.Key == "" || !strings.HasPrefix(c.Value, "/") {
			continue
		}
		if strings.Contains(canonical, "*"+c.Key) {
			c.Value = strings.TrimPrefix(c.Value, "/")
		}
	}
}
