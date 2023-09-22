package base

import "context"

type Nexter[T any] struct {
	index int
	chain Chain[T]
}

func (n Nexter[T]) Next(c context.Context, req any, info *T) (resp any, err error) {
	if n.index >= len(n.chain) {
		return nil, nil
	}
	return n.chain[n.index].Handle(c, req, n.newNext(), info)
}

func (n Nexter[T]) newNext() Nexter[T] {
	return Nexter[T]{
		index: n.index + 1,
		chain: n.chain,
	}
}

type Interceptor[T any] interface {
	Handle(c context.Context, req any, next Nexter[T], info *T) (any, error)
}

type Chain[T any] []Interceptor[T]

func (chain Chain[T]) Do(c context.Context, req any, info *T) (resp any, err error) {
	next := Nexter[T]{
		index: 0,
		chain: chain,
	}
	return next.Next(c, req, info)
}
