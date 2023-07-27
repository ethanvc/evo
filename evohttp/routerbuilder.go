package evohttp

import (
	"net/http"
	"path"
)

type RouterBuilder struct {
	router   *Router
	handlers HandlerChain
	basePath string
}

func NewRouterBuilder() *RouterBuilder {
	b := &RouterBuilder{
		router: NewRouter(),
	}
	return b
}

func (b *RouterBuilder) Use(middleware ...Handler) {
	b.handlers = append(b.handlers, middleware...)
}

func (b *RouterBuilder) UseF(middleware ...HandlerFunc) {
	b.Use(funcToHandlers(middleware)...)
}

func (b *RouterBuilder) SubBuilder(relativePath string, handlers ...Handler) *RouterBuilder {
	child := &RouterBuilder{
		router: b.router,
	}
	child.handlers = joinSlice(b.handlers, handlers)
	child.basePath = path.Join(b.basePath, relativePath)
	return child
}

func (b *RouterBuilder) POST(relativePath string, handlers ...Handler) {
	b.router.addRoute(http.MethodPost, path.Join(b.basePath, relativePath), handlers)
}

func (b *RouterBuilder) POSTF(relativePath string, handlers ...HandlerFunc) {
	b.POST(relativePath, funcToHandlers(handlers)...)
}

func (b *RouterBuilder) GET(relativePath string, handlers ...Handler) {
	b.router.addRoute(http.MethodGet, path.Join(b.basePath, relativePath), handlers)
}

func (b *RouterBuilder) GETF(relativePath string, handlers ...HandlerFunc) {
	b.GET(relativePath, funcToHandlers(handlers)...)
}

func funcToHandlers(h []HandlerFunc) []Handler {
	var hs []Handler
	for _, v := range h {
		hs = append(hs, HandlerFunc(v))
	}
	return hs
}

func (b *RouterBuilder) handle(httpMethod, relativePath string, handlers HandlerChain) {
	urlPath := path.Join(b.basePath, relativePath)
	handlers = joinSlice(b.handlers, handlers)
	b.router.addRoute(httpMethod, urlPath, handlers)
}

func joinSlice[T any](a ...[]T) []T {
	var x []T
	for _, v := range a {
		x = append(x, v...)
	}
	return x
}
