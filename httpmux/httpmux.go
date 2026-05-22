// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

// Package httpmux is a generic trie based HTTP path mux.
package httpmux

import (
	"net/http"
	"sync"
)

// Param is a single URL parameter, consisting of a key and a value.
type Param struct {
	Key   string
	Value string
}

// Params is a Param-slice, as returned by the mux.
// The slice is ordered, the first URL parameter is also the first slice value.
// It is therefore safe to read values by the index.
type Params []Param

// ByName returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) ByName(name string) string {
	for _, p := range ps {
		if p.Key == name {
			return p.Value
		}
	}
	return ""
}

const defaultParamsCap = 16

var paramsPool sync.Pool

func putParams(ps *Params) {
	if ps != nil {
		*ps = (*ps)[:0]
		paramsPool.Put(ps)
	}
}

// PutParams returns ps to the global params pool. ps must be a non-nil pointer
// previously returned by Lookup and must not be used after PutParams.
func PutParams(ps *Params) {
	putParams(ps)
}

// HttpMux is a generic radix-tree mux keyed by HTTP method and path pattern.
type HttpMux[T any] struct {
	trees map[string]*Node[T]

	maxParams uint16
}

// New returns a new initialized HttpMux.
func New[T any]() *HttpMux[T] {
	return &HttpMux[T]{}
}

func (r *HttpMux[T]) getParams() *Params {
	v := paramsPool.Get()
	var ps *Params
	if v == nil {
		ps = r.newParams()
	} else {
		ps = v.(*Params)
	}

	wantCap := r.maxParams
	if wantCap < defaultParamsCap {
		wantCap = defaultParamsCap
	}
	if cap(*ps) < int(wantCap) {
		*ps = make(Params, 0, wantCap)
	} else {
		*ps = (*ps)[:0]
	}
	return ps
}

func (r *HttpMux[T]) newParams() *Params {
	capacity := r.maxParams
	if capacity < defaultParamsCap {
		capacity = defaultParamsCap
	}
	ps := make(Params, 0, capacity)
	return &ps
}

// GET is a shortcut for mux.Register(http.MethodGet, path, value).
func (r *HttpMux[T]) GET(path string, value T) {
	r.Register(http.MethodGet, path, value)
}

// HEAD is a shortcut for mux.Register(http.MethodHead, path, value).
func (r *HttpMux[T]) HEAD(path string, value T) {
	r.Register(http.MethodHead, path, value)
}

// OPTIONS is a shortcut for mux.Register(http.MethodOptions, path, value).
func (r *HttpMux[T]) OPTIONS(path string, value T) {
	r.Register(http.MethodOptions, path, value)
}

// POST is a shortcut for mux.Register(http.MethodPost, path, value).
func (r *HttpMux[T]) POST(path string, value T) {
	r.Register(http.MethodPost, path, value)
}

// PUT is a shortcut for mux.Register(http.MethodPut, path, value).
func (r *HttpMux[T]) PUT(path string, value T) {
	r.Register(http.MethodPut, path, value)
}

// PATCH is a shortcut for mux.Register(http.MethodPatch, path, value).
func (r *HttpMux[T]) PATCH(path string, value T) {
	r.Register(http.MethodPatch, path, value)
}

// DELETE is a shortcut for mux.Register(http.MethodDelete, path, value).
func (r *HttpMux[T]) DELETE(path string, value T) {
	r.Register(http.MethodDelete, path, value)
}

// Handle registers a value for method + path.
func (r *HttpMux[T]) Handle(method, path string, value T) {
	r.Register(method, path, value)
}

// Register stores value for method + path.
func (r *HttpMux[T]) Register(method, path string, value T) {
	if method == "" {
		panic("method must not be empty")
	}
	if len(path) < 1 || path[0] != '/' {
		panic("path must begin with '/' in path '" + path + "'")
	}

	if r.trees == nil {
		r.trees = make(map[string]*Node[T])
	}

	root := r.trees[method]
	if root == nil {
		root = new(Node[T])
		r.trees[method] = root
	}

	root.addRoute(path, value)

	if paramsCount := countParams(path); paramsCount > r.maxParams {
		r.maxParams = paramsCount
	}
}

// Lookup returns the node registered for method + path and any captured params.
// When params is non-nil, the caller must call PutParams(params) after use.
// The third return value indicates whether a trailing-slash redirect is recommended.
func (r *HttpMux[T]) Lookup(method, path string) (*Node[T], *Params, bool) {
	if root := r.trees[method]; root != nil {
		match, ps, tsr := root.getValue(path, r.getParams)
		if match == nil {
			putParams(ps)
			return nil, nil, tsr
		}
		return match, ps, tsr
	}
	return nil, nil, false
}
