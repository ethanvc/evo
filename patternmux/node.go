package patternmux

func (n *Node[T]) Raw() string {
	return n.raw
}

func (n *Node[T]) Canonical() string {
	return n.canonical
}

func (n *Node[T]) HasKeep() bool {
	return n.hasKeep
}

func (n *Node[T]) CachedConverted() string {
	return n.cachedConverted
}
