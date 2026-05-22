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

// Search for a wildcard segment and check the name for invalid characters.
// Returns -1 as index, if no wildcard was found.
func findWildcard(path string) (wildcard string, i int, valid bool) {
	// Find start
	for start, c := range []byte(path) {
		// A wildcard starts with ':' (param) or '*' (catch-all)
		if c != ':' && c != '*' {
			continue
		}

		// Find end and check for invalid characters
		valid = true
		for end, c := range []byte(path[start+1:]) {
			switch c {
			case '/':
				return path[start : start+1+end], start, valid
			case ':', '*':
				valid = false
			}
		}
		return path[start:], start, valid
	}
	return "", -1, false
}

func countParams(path string) uint16 {
	var n uint16
	for i := range []byte(path) {
		switch path[i] {
		case ':', '*':
			n++
		}
	}
	return n
}

type nodeType uint8

const (
	static nodeType = iota // default
	root
	param
	catchAll
)

type routeMeta struct {
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

	// Radix-tree internals.
	path       string
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

// childIndex maps an indices position to the children slice index.
// When wildChild is set, children[0] is the wildcard node and static children follow.
func (n *Node[T]) childIndex(i int) int {
	if n.wildChild {
		return i + 1
	}
	return i
}

// Increments priority of the given child and reorders if necessary.
func (n *Node[T]) incrementChildPrio(pos int) int {
	ci := n.childIndex(pos)
	cs := n.children
	cs[ci].priority++
	prio := cs[ci].priority

	// Adjust position (move to front)
	newPos := pos
	for ; newPos > 0 && cs[n.childIndex(newPos-1)].priority < prio; newPos-- {
		a, b := n.childIndex(newPos-1), n.childIndex(newPos)
		cs[a], cs[b] = cs[b], cs[a]
	}

	// Build new index char string
	if newPos != pos {
		n.indices = n.indices[:newPos] + // Unchanged prefix, might be empty
			n.indices[pos:pos+1] + // The index char we move
			n.indices[newPos:pos] + n.indices[pos+1:] // Rest without char at 'pos'
	}

	return newPos
}

// addRoute adds a node with the given value to the path.
// Not concurrency-safe.
func (n *Node[T]) addRoute(path string, value T, meta routeMeta) {
	fullPath := path
	n.priority++

	// Empty tree
	if n.path == "" && n.indices == "" {
		n.insertChild(path, fullPath, value, meta)
		n.nType = root
		return
	}

walk:
	for {
		// Find the longest common prefix.
		// This also implies that the common prefix contains no ':' or '*'
		// since the existing key can't contain those chars.
		i := longestCommonPrefix(path, n.path)

		// Split edge
		if i < len(n.path) {
			child := Node[T]{
				raw:             n.raw,
				canonical:       n.canonical,
				hasKeep:         n.hasKeep,
				cachedConverted: n.cachedConverted,
				literalPrefix:   n.literalPrefix,
				registerOrder:   n.registerOrder,

				path:       n.path[i:],
				wildChild:  n.wildChild,
				nType:      static,
				indices:    n.indices,
				children:   n.children,
				value:      n.value,
				registered: n.registered,
				priority:   n.priority - 1,
			}

			n.children = []*Node[T]{&child}
			// []byte for proper unicode char conversion, see #65
			n.indices = string([]byte{n.path[i]})
			n.path = path[:i]
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

		// Make new node a child of this node
		if i < len(path) {
			path = path[i:]

			if n.wildChild {
				wc := n.children[0]
				wc.priority++

				// Continue into wildcard only when the remaining path starts with the wildcard token.
				if len(path) >= len(wc.path) && wc.path == path[:len(wc.path)] &&
					wc.nType != catchAll &&
					(len(wc.path) >= len(path) || path[len(wc.path)] == '/') {
					n = wc
					continue walk
				}

				if path[0] == ':' || path[0] == '*' {
					pathSeg := path
					if wc.nType != catchAll {
						pathSeg = strings.SplitN(pathSeg, "/", 2)[0]
					}
					prefix := fullPath[:strings.Index(fullPath, pathSeg)] + wc.path
					panic("'" + pathSeg +
						"' in new path '" + fullPath +
						"' conflicts with existing wildcard '" + wc.path +
						"' in existing prefix '" + prefix +
						"'")
				}
			}

			idxc := path[0]

			// '/' after param
			if n.nType == param && idxc == '/' && len(n.children) == 1 {
				n = n.children[0]
				n.priority++
				continue walk
			}

			// Check if a child with the next path byte exists
			for i, c := range []byte(n.indices) {
				if c == idxc {
					i = n.incrementChildPrio(i)
					n = n.children[n.childIndex(i)]
					continue walk
				}
			}

			// Otherwise insert it
			if idxc != ':' && idxc != '*' {
				// []byte for proper unicode char conversion, see #65
				n.indices += string([]byte{idxc})
				child := &Node[T]{}
				n.children = append(n.children, child)
				n.incrementChildPrio(len(n.indices) - 1)
				n = child
			}
			n.insertChild(path, fullPath, value, meta)
			return
		}

		// Otherwise add value to current node
		if n.registered {
			panic("a value is already registered for path '" + fullPath + "'")
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

func (n *Node[T]) insertChild(path, fullPath string, value T, meta routeMeta) {
	for {
		// Find prefix until first wildcard
		wildcard, i, valid := findWildcard(path)
		if i < 0 { // No wildcard found
			break
		}

		// The wildcard name must not contain ':' and '*'
		if !valid {
			panic("only one wildcard per path segment is allowed, has: '" +
				wildcard + "' in path '" + fullPath + "'")
		}

		// Check if the wildcard has a name
		if len(wildcard) < 2 {
			panic("wildcards must be named with a non-empty name in path '" + fullPath + "'")
		}

		// Check if this node has existing children which would be
		// unreachable if we insert the wildcard here
		if len(n.children) > 0 {
			panic("wildcard segment '" + wildcard +
				"' conflicts with existing children in path '" + fullPath + "'")
		}

		// param
		if wildcard[0] == ':' {
			if i > 0 {
				// Insert prefix before the current wildcard
				n.path = path[:i]
				path = path[i:]
			}

			n.wildChild = true
			child := &Node[T]{
				nType: param,
				path:  wildcard,
			}
			n.children = []*Node[T]{child}
			n = child
			n.priority++

			// If the path doesn't end with the wildcard, then there
			// will be another non-wildcard subpath starting with '/'
			if len(wildcard) < len(path) {
				path = path[len(wildcard):]
				child := &Node[T]{
					priority: 1,
				}
				n.children = []*Node[T]{child}
				n = child
				continue
			}

			// Otherwise we're done. Insert the value in the new leaf.
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

		// catchAll
		if i+len(wildcard) != len(path) {
			panic("catch-all routes are only allowed at the end of the path in path '" + fullPath + "'")
		}

		if len(n.path) > 0 && n.path[len(n.path)-1] == '/' {
			panic("catch-all conflicts with existing value for the path segment root in path '" + fullPath + "'")
		}

		// Currently fixed width 1 for '/'
		i--
		if path[i] != '/' {
			panic("no / before catch-all in path '" + fullPath + "'")
		}

		n.path = path[:i]

		// First node: catchAll node with empty path
		child := &Node[T]{
			wildChild: true,
			nType:     catchAll,
		}
		n.children = []*Node[T]{child}
		n.indices = string('/')
		n = child
		n.priority++

		// Second node: node holding the variable
		child = &Node[T]{
			path:       path[i:],
			nType:      catchAll,
			value:      value,
			registered: true,
			priority:   1,
			raw:        meta.raw,
			canonical:  meta.canonical,

			hasKeep:         meta.hasKeep,
			cachedConverted: meta.cachedConverted,
			literalPrefix:   meta.literalPrefix,
			registerOrder:   meta.registerOrder,
		}
		n.children = []*Node[T]{child}

		return
	}

	// If no wildcard was found, simply insert the path and value.
	n.path = path
	n.value = value
	n.registered = true
	n.raw = meta.raw
	n.canonical = meta.canonical
	n.hasKeep = meta.hasKeep
	n.cachedConverted = meta.cachedConverted
	n.literalPrefix = meta.literalPrefix
	n.registerOrder = meta.registerOrder
}

// getValue returns the value registered with the given path.
// Captures are allocated lazily using alloc when wildcard values are found.
func (n *Node[T]) getValue(path string, alloc func() *Captures) (match *Node[T], caps *Captures, ok bool) {
walk: // Outer loop for walking the tree
	for {
		prefix := n.path
		if len(path) > len(prefix) {
			if path[:len(prefix)] == prefix {
				path = path[len(prefix):]

				idxc := path[0]
				for i, c := range []byte(n.indices) {
					if c == idxc {
						n = n.children[n.childIndex(i)]
						continue walk
					}
				}

				if !n.wildChild {
					return nil, caps, false
				}

				// Handle wildcard child
				n = n.children[0]
				switch n.nType {
				case param:
					// Find param end (either '/' or path end)
					end := 0
					for end < len(path) && path[end] != '/' {
						end++
					}

					// Save param value
					if alloc != nil {
						if caps == nil {
							caps = alloc()
						}
						// Expand slice within preallocated capacity
						i := len(*caps)
						*caps = (*caps)[:i+1]
						(*caps)[i] = Capture{
							Key:   n.path[1:],
							Value: path[:end],
						}
					}

					// We need to go deeper.
					if end < len(path) {
						if len(n.children) > 0 {
							path = path[end:]
							n = n.children[0]
							continue walk
						}
						return nil, caps, false
					}

					if n.registered {
						return n, caps, true
					}
					return nil, caps, false

				case catchAll:
					// Save catch-all value
					if alloc != nil {
						if caps == nil {
							caps = alloc()
						}
						// Expand slice within preallocated capacity
						i := len(*caps)
						*caps = (*caps)[:i+1]
						(*caps)[i] = Capture{
							Key:   n.path[2:],
							Value: path,
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
		} else if path == prefix {
			// We should have reached the node containing the value.
			if n.registered {
				return n, caps, true
			}
			return nil, caps, false
		}

		return nil, caps, false
	}
}
