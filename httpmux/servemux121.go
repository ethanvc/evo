// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpmux

// This file implements ServeMux behavior as in Go 1.21.
// The behavior is controlled by a GODEBUG setting.
// Most of this code is derived from commit 08e35cc334.
// Changes are minimal: aside from the different receiver type,
// they mostly involve renaming functions, usually by unexporting them.

import (
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
)

var use121 bool

// Read httpmuxgo121 once at startup, since dealing with changes to it during
// program execution is too complex and error-prone.
func init() {
	use121 = godebugHTTPMuxGo121()
}

func godebugHTTPMuxGo121() bool {
	for _, field := range strings.Split(os.Getenv("GODEBUG"), ",") {
		if strings.TrimSpace(field) == "httpmuxgo121=1" {
			return true
		}
	}
	return false
}

// serveMux121 holds the state of a ServeMux needed for Go 1.21 behavior.
type serveMux121 struct {
	mu    sync.RWMutex
	m     map[string]muxEntry
	es    []muxEntry // slice of entries sorted from longest to shortest.
	hosts bool       // whether any patterns contain hostnames
}

type muxEntry struct {
	h       Handler
	pattern string
}

// Formerly ServeMux.Handle.
func (mux *serveMux121) handle(pattern string, handler Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	if pattern == "" {
		panic("http: invalid pattern")
	}
	if handler == nil {
		panic("http: nil handler")
	}
	if _, exist := mux.m[pattern]; exist {
		panic("http: multiple registrations for " + pattern)
	}

	if mux.m == nil {
		mux.m = make(map[string]muxEntry)
	}
	e := muxEntry{h: handler, pattern: pattern}
	mux.m[pattern] = e
	if pattern[len(pattern)-1] == '/' {
		mux.es = appendSorted(mux.es, e)
	}

	if pattern[0] != '/' {
		mux.hosts = true
	}
}

func appendSorted(es []muxEntry, e muxEntry) []muxEntry {
	n := len(es)
	i := sort.Search(n, func(i int) bool {
		return len(es[i].pattern) < len(e.pattern)
	})
	if i == n {
		return append(es, e)
	}
	es = append(es, muxEntry{})
	copy(es[i+1:], es[i:])
	es[i] = e
	return es
}

// Formerly ServeMux.HandleFunc.
func (mux *serveMux121) handleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	if handler == nil {
		panic("http: nil handler")
	}
	mux.handle(pattern, HandlerFunc(handler))
}

// Formerly ServeMux.Handler.
func (mux *serveMux121) findHandler(r *Request) (h Handler, pattern string) {
	if r.Method == "CONNECT" {
		if u, ok := mux.redirectToPathSlash(r.URL.Host, r.URL.Path, r.URL); ok {
			return RedirectHandler(u.String(), StatusMovedPermanently), u.Path
		}

		return mux.handler(r.Host, r.URL.Path)
	}

	host := stripHostPort(r.Host)
	path := cleanPath(r.URL.Path)

	if u, ok := mux.redirectToPathSlash(host, path, r.URL); ok {
		return RedirectHandler(u.String(), StatusMovedPermanently), u.Path
	}

	if path != r.URL.Path {
		_, pattern = mux.handler(host, path)
		u := &url.URL{Path: path, RawQuery: r.URL.RawQuery}
		return RedirectHandler(u.String(), StatusMovedPermanently), pattern
	}

	return mux.handler(host, r.URL.Path)
}

// handler is the main implementation of findHandler.
func (mux *serveMux121) handler(host, path string) (h Handler, pattern string) {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	if mux.hosts {
		h, pattern = mux.match(host + path)
	}
	if h == nil {
		h, pattern = mux.match(path)
	}
	if h == nil {
		h, pattern = NotFoundHandler(), ""
	}
	return
}

// Find a handler on a handler map given a path string.
func (mux *serveMux121) match(path string) (h Handler, pattern string) {
	v, ok := mux.m[path]
	if ok {
		return v.h, v.pattern
	}

	for _, e := range mux.es {
		if strings.HasPrefix(path, e.pattern) {
			return e.h, e.pattern
		}
	}
	return nil, ""
}

// redirectToPathSlash determines if the given path needs appending "/" to it.
func (mux *serveMux121) redirectToPathSlash(host, path string, u *url.URL) (*url.URL, bool) {
	mux.mu.RLock()
	shouldRedirect := mux.shouldRedirectRLocked(host, path)
	mux.mu.RUnlock()
	if !shouldRedirect {
		return u, false
	}
	path = path + "/"
	u = &url.URL{Path: path, RawQuery: u.RawQuery}
	return u, true
}

// shouldRedirectRLocked reports whether the given path and host should be redirected to
// path+"/".
func (mux *serveMux121) shouldRedirectRLocked(host, path string) bool {
	p := []string{path, host + path}

	for _, c := range p {
		if _, exist := mux.m[c]; exist {
			return false
		}
	}

	n := len(path)
	if n == 0 {
		return false
	}
	for _, c := range p {
		if _, exist := mux.m[c+"/"]; exist {
			return path[n-1] != '/'
		}
	}

	return false
}
