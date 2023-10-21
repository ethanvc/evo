package base

import "container/heap"

type Heap[T any] struct {
	x heapInternal[T]
}

func NewHeap[T any](less func(t0, t1 T) bool, swap func(t0, t1 T, i, j int)) *Heap[T] {
	if less == nil {
		panic("less must not be nil")
	}
	return &Heap[T]{
		x: heapInternal[T]{
			less: less,
			swap: swap,
		},
	}
}

func (h *Heap[T]) Init(x []T) {
	h.x.t = x
	heap.Init(&h.x)
}

func (h *Heap[T]) Fix(i int) {
	heap.Fix(&h.x, i)
}

func (h *Heap[T]) Push(v T) {
	heap.Push(&h.x, v)
}

func (h *Heap[T]) Pop() T {
	v := heap.Pop(&h.x)
	return v.(T)
}

func (h *Heap[T]) Remove(i int) T {
	v := heap.Remove(&h.x, i)
	return v.(T)
}

type heapInternal[T any] struct {
	t    []T
	less func(t0, t1 T) bool
	swap func(t0, t1 T, i, j int)
}

func (h *heapInternal[T]) Len() int {
	return len(h.t)
}

func (h *heapInternal[T]) Less(i, j int) bool {
	return h.less(h.t[i], h.t[j])
}

func (h *heapInternal[T]) Swap(i, j int) {
	h.t[i], h.t[j] = h.t[j], h.t[i]
	if h.swap != nil {
		h.swap(h.t[i], h.t[j], j, i)
	}
}

func (h *heapInternal[T]) Push(x any) {
	h.t = append(h.t, x)
}

func (h *heapInternal[T]) Pop() any {
	n := len(h.t)
	old := h.t[n-1]
	h.t = h.t[0 : n-1]
	return old
}
