package patternmux

import "strings"

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func longestCommonPrefix(a, b string) int {
	i := 0
	max := min(len(a), len(b))
	for i < max && a[i] == b[i] {
		i++
	}
	return i
}

// findWildcard locates the next lowered wildcard in an index key.
func findWildcard(key string) (token string, i int, isRest bool, valid bool) {
	for start := 0; start+1 < len(key); start++ {
		if key[start] != '\x00' {
			continue
		}
		switch key[start+1] {
		case 'S':
			isRest = false
		case 'R':
			isRest = true
		default:
			continue
		}

		valid = true
		for end, c := range []byte(key[start+2:]) {
			switch c {
			case untilSlashBoundary:
				return key[start : start+2+end], start, isRest, valid
			case '\x00':
				valid = false
			}
		}
		return key[start:], start, isRest, valid
	}
	return "", -1, false, false
}

func countIndexWildcards(key string) uint16 {
	var n uint16
	for i := 0; i+1 < len(key); i++ {
		if key[i] != '\x00' {
			continue
		}
		switch key[i+1] {
		case 'S', 'R':
			n++
		}
	}
	return n
}

func isWildcardMark(b byte) bool {
	return b == '\x00'
}

type nodeType uint8

const (
	static nodeType = iota
	root
	untilSlashNode
	restNode
)

type patternMeta struct {
	raw             string
	canonical       string
	hasKeep         bool
	cachedConverted string
	literalPrefix   int
	registerOrder   uint64
}

type Node[T any] struct {
	// Registration metadata.
	raw             string
	canonical       string
	hasKeep         bool
	cachedConverted string
	literalPrefix   int
	registerOrder   uint64

	// Radix index internals.
	prefix     string
	captureKey string
	indices    string
	wildChild  bool
	nType      nodeType
	priority   uint32
	children   []*Node[T]
	value      T
	registered bool
}

// Value returns the value registered on this node.
func (n *Node[T]) Value() T {
	return n.value
}

func (n *Node[T]) childIndex(i int) int {
	if n.wildChild {
		return i + 1
	}
	return i
}

func (n *Node[T]) incrementChildPrio(pos int) int {
	ci := n.childIndex(pos)
	cs := n.children
	cs[ci].priority++
	prio := cs[ci].priority

	newPos := pos
	for ; newPos > 0 && cs[n.childIndex(newPos-1)].priority < prio; newPos-- {
		a, b := n.childIndex(newPos-1), n.childIndex(newPos)
		cs[a], cs[b] = cs[b], cs[a]
	}

	if newPos != pos {
		n.indices = n.indices[:newPos] +
			n.indices[pos:pos+1] +
			n.indices[newPos:pos] + n.indices[pos+1:]
	}

	return newPos
}

// insertIndexed adds a lowered pattern key into the radix index.
func (n *Node[T]) insertIndexed(key string, value T, meta patternMeta) {
	fullKey := key
	n.priority++

	if n.prefix == "" && n.indices == "" {
		n.insertChild(key, fullKey, value, meta)
		n.nType = root
		return
	}

walk:
	for {
		i := longestCommonPrefix(key, n.prefix)

		if i < len(n.prefix) {
			child := Node[T]{
				raw:             n.raw,
				canonical:       n.canonical,
				hasKeep:         n.hasKeep,
				cachedConverted: n.cachedConverted,
				literalPrefix:   n.literalPrefix,
				registerOrder:   n.registerOrder,

				prefix:     n.prefix[i:],
				wildChild:  n.wildChild,
				nType:      static,
				indices:    n.indices,
				children:   n.children,
				value:      n.value,
				registered: n.registered,
				priority:   n.priority - 1,
			}

			n.children = []*Node[T]{&child}
			n.indices = string([]byte{n.prefix[i]})
			n.prefix = key[:i]
			n.value = *new(T)
			n.registered = false
			n.wildChild = false

			n.raw = ""
			n.canonical = ""
			n.hasKeep = false
			n.cachedConverted = ""
			n.literalPrefix = 0
			n.registerOrder = 0
		}

		if i < len(key) {
			key = key[i:]

			if n.wildChild {
				wc := n.children[0]
				wc.priority++

				if len(key) >= len(wc.prefix) && wc.prefix == key[:len(wc.prefix)] &&
					wc.nType != restNode &&
					(len(wc.prefix) >= len(key) || key[len(wc.prefix)] == untilSlashBoundary) {
					n = wc
					continue walk
				}

				if isWildcardMark(key[0]) {
					keySeg := key
					if wc.nType != restNode {
						keySeg = segmentUntilBoundary(keySeg, untilSlashBoundary)
					}
					prefix := fullKey[:strings.Index(fullKey, keySeg)] + wc.prefix
					panic("'" + keySeg +
						"' in new pattern '" + fullKey +
						"' conflicts with existing wildcard '" + wc.prefix +
						"' in existing prefix '" + prefix +
						"'")
				}
			}

			idxc := key[0]

			if n.nType == untilSlashNode && idxc == untilSlashBoundary && len(n.children) == 1 {
				n = n.children[0]
				n.priority++
				continue walk
			}

			for i, c := range []byte(n.indices) {
				if c == idxc {
					i = n.incrementChildPrio(i)
					n = n.children[n.childIndex(i)]
					continue walk
				}
			}

			if !isWildcardMark(idxc) {
				n.indices += string([]byte{idxc})
				child := &Node[T]{}
				n.children = append(n.children, child)
				n.incrementChildPrio(len(n.indices) - 1)
				n = child
			}
			n.insertChild(key, fullKey, value, meta)
			return
		}

		if n.registered {
			panic("a value is already registered for pattern '" + fullKey + "'")
		}
		n.value = value
		n.registered = true
		n.raw = meta.raw
		n.canonical = meta.canonical
		n.hasKeep = meta.hasKeep
		n.cachedConverted = meta.cachedConverted
		n.literalPrefix = meta.literalPrefix
		n.registerOrder = meta.registerOrder
		return
	}
}

func (n *Node[T]) insertChild(key, fullKey string, value T, meta patternMeta) {
	for {
		token, i, isRest, valid := findWildcard(key)
		if i < 0 {
			break
		}

		if !valid {
			panic("only one wildcard per until-slash segment is allowed, has: '" +
				token + "' in pattern '" + fullKey + "'")
		}

		if len(token) < 3 {
			panic("wildcards must be named with a non-empty name in pattern '" + fullKey + "'")
		}

		if len(n.children) > 0 {
			panic("wildcard segment '" + token +
				"' conflicts with existing children in pattern '" + fullKey + "'")
		}

		captureKey := token[2:]

		if !isRest {
			if i > 0 {
				n.prefix = key[:i]
				key = key[i:]
			}

			n.wildChild = true
			child := &Node[T]{
				nType:      untilSlashNode,
				prefix:     token,
				captureKey: captureKey,
			}
			n.children = []*Node[T]{child}
			n = child
			n.priority++

			if len(token) < len(key) {
				key = key[len(token):]
				child := &Node[T]{
					priority: 1,
				}
				n.children = []*Node[T]{child}
				n = child
				continue
			}

			n.value = value
			n.registered = true
			n.raw = meta.raw
			n.canonical = meta.canonical
			n.hasKeep = meta.hasKeep
			n.cachedConverted = meta.cachedConverted
			n.literalPrefix = meta.literalPrefix
			n.registerOrder = meta.registerOrder
			return
		}

		if i+len(token) != len(key) {
			panic("rest rule expression must be at end of pattern '" + fullKey + "'")
		}

		if i > 0 {
			n.prefix = key[:i]
		}

		n.wildChild = true
		child := &Node[T]{
			nType:           restNode,
			prefix:          token,
			captureKey:      captureKey,
			value:           value,
			registered:      true,
			priority:        1,
			raw:             meta.raw,
			canonical:       meta.canonical,
			hasKeep:         meta.hasKeep,
			cachedConverted: meta.cachedConverted,
			literalPrefix:   meta.literalPrefix,
			registerOrder:   meta.registerOrder,
		}
		n.children = []*Node[T]{child}
		return
	}

	n.prefix = key
	n.value = value
	n.registered = true
	n.raw = meta.raw
	n.canonical = meta.canonical
	n.hasKeep = meta.hasKeep
	n.cachedConverted = meta.cachedConverted
	n.literalPrefix = meta.literalPrefix
	n.registerOrder = meta.registerOrder
}

func segmentUntilBoundary(s string, boundary byte) string {
	if i := strings.IndexByte(s, boundary); i >= 0 {
		return s[:i]
	}
	return s
}

// matchInput walks the radix index for input against registered patterns.
func (n *Node[T]) matchInput(input string, alloc func() *Captures) (match *Node[T], caps *Captures, ok bool) {
walk:
	for {
		prefix := n.prefix
		if len(input) > len(prefix) {
			if input[:len(prefix)] == prefix {
				input = input[len(prefix):]

				idxc := input[0]
				for i, c := range []byte(n.indices) {
					if c == idxc {
						n = n.children[n.childIndex(i)]
						continue walk
					}
				}

				if !n.wildChild {
					return nil, caps, false
				}

				n = n.children[0]
				switch n.nType {
				case untilSlashNode:
					end := strings.IndexByte(input, untilSlashBoundary)
					if end < 0 {
						end = len(input)
					}

					if alloc != nil {
						if caps == nil {
							caps = alloc()
						}
						i := len(*caps)
						*caps = (*caps)[:i+1]
						(*caps)[i] = Capture{
							Key:   n.captureKey,
							Value: input[:end],
						}
					}

					if end < len(input) {
						if len(n.children) > 0 {
							input = input[end:]
							n = n.children[0]
							continue walk
						}
						return nil, caps, false
					}

					if n.registered {
						return n, caps, true
					}
					return nil, caps, false

				case restNode:
					if alloc != nil {
						if caps == nil {
							caps = alloc()
						}
						i := len(*caps)
						*caps = (*caps)[:i+1]
						(*caps)[i] = Capture{
							Key:   n.captureKey,
							Value: input,
						}
					}

					if n.registered {
						return n, caps, true
					}
					return nil, caps, false

				default:
					panic("invalid node type")
				}
			}
		} else if input == prefix {
			if n.registered {
				return n, caps, true
			}
			return nil, caps, false
		}

		return nil, caps, false
	}
}
