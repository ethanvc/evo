package base

import "context"

type Interceptor[T any] interface {
	Handle(c context.Context, req any, info T, next Nexter[T]) (any, error)
}

type Nexter[T any] struct {
	index int
	chain Chain[T]
}

func (n Nexter[T]) newNext() Nexter[T] {
	return Nexter[T]{
		index: n.index + 1,
		chain: n.chain,
	}
}

func (n Nexter[T]) LastHandler() Interceptor[T] {
	l := len(n.chain)
	if l == 0 {
		return nil
	} else {
		return n.chain[l-1]
	}
}

func (n Nexter[T]) Next(c context.Context, req any, info T) (resp any, err error) {
	if n.index >= len(n.chain) {
		return nil, nil
	}
	return n.chain[n.index].Handle(c, req, info, n.newNext())
}

type Chain[T any] []Interceptor[T]

func (chain Chain[T]) Do(c context.Context, req any, info T) (resp any, err error) {
	next := Nexter[T]{
		index: 0,
		chain: chain,
	}
	return next.Next(c, req, info)
}

type InterceptorFunc[T any] func(c context.Context, req any, info T, nexter Nexter[T]) (any, error)

func (h InterceptorFunc[T]) Handle(c context.Context, req any, info T, next Nexter[T]) (resp any, err error) {
	return h(c, req, info, next)
}
