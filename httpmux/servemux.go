// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpmux

import (
	"errors"
	"fmt"
	"maps"
	"net"
	"net/url"
	"path"
	"runtime"
	"slices"
	"strings"
	"sync"

	"golang.org/x/net/http/httpguts"
)

// A ServeMux is an HTTP request multiplexer.
//
// ServeMux matches the URL of each incoming request against a list of
// registered patterns and calls the handler for the pattern that most closely
// matches the URL.
type ServeMux struct {
	mu    sync.RWMutex
	tree  routingNode
	index routingIndex
}

// NewServeMux allocates and returns a new [ServeMux].
func NewServeMux() *ServeMux {
	return &ServeMux{}
}

// DefaultServeMux is the default [ServeMux].
var DefaultServeMux = &defaultServeMux

var defaultServeMux ServeMux

// cleanPath returns the canonical path for p, eliminating . and .. elements.
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	return trimTrailingSlash(path.Clean(p))
}

func trimTrailingSlash(p string) string {
	for len(p) > 1 && p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}
	return p
}

// stripHostPort returns h without any trailing ":<port>".
func stripHostPort(h string) string {
	// If no port on host, return unchanged
	if !strings.Contains(h, ":") {
		return h
	}
	host, _, err := net.SplitHostPort(h)
	if err != nil {
		return h // on error, return unchanged
	}
	return host
}

// Handler returns the handler to use for the given request,
// consulting r.Method, r.Host, and r.URL.Path. It always returns
// a non-nil handler.
//
// Handler does not modify its argument. In particular, it does not populate
// named path wildcards, so r.PathValue will always return the empty string.
func (mux *ServeMux) Handler(r *Request) (h Handler, pattern string) {
	h, p, _, _ := mux.findHandler(r.Method, r.Host, r.URL.EscapedPath(), r.URL.RawQuery)
	return h, p
}

// findHandler finds a handler for a request.
func (mux *ServeMux) findHandler(method, host, escapedPath, rawQuery string) (h Handler, patStr string, _ *pattern, matches []string) {
	var n *routingNode
	path := escapedPath
	// CONNECT requests are not canonicalized.
	if method == "CONNECT" {
		path = trimTrailingSlash(path)
		n, matches = mux.match(host, method, path)
	} else {
		host = stripHostPort(host)
		path = cleanPath(path)

		n, matches = mux.match(host, method, path)
		if path != escapedPath && path != trimTrailingSlash(escapedPath) {
			patStr := ""
			if n != nil {
				patStr = n.pattern.String()
			}
			u := &url.URL{Path: path, RawQuery: rawQuery}
			return RedirectHandler(u.String(), StatusTemporaryRedirect), patStr, nil, nil
		}
	}
	if n == nil {
		allowedMethods := mux.matchingMethods(host, path)
		if len(allowedMethods) > 0 {
			return HandlerFunc(func(w ResponseWriter, r *Request) {
				w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
				Error(w, StatusText(StatusMethodNotAllowed), StatusMethodNotAllowed)
			}), "", nil, nil
		}
		return NotFoundHandler(), "", nil, nil
	}
	return n.handler, n.pattern.String(), n.pattern, matches
}

// match looks up a node in the tree that matches the host, method and path.
func (mux *ServeMux) match(host, method, path string) (_ *routingNode, matches []string) {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	n, matches := mux.tree.match(host, method, path)
	return n, matches
}

// matchingMethods return a sorted list of all methods that would match with the given host and path.
func (mux *ServeMux) matchingMethods(host, path string) []string {
	mux.mu.RLock()
	defer mux.mu.RUnlock()
	ms := map[string]bool{}
	mux.tree.matchingMethods(host, path, ms)
	return slices.Sorted(maps.Keys(ms))
}

// ServeHTTP dispatches the request to the handler whose pattern most closely
// matches the request URL.
func (mux *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
	if r.RequestURI == "*" {
		if r.ProtoAtLeast(1, 1) {
			w.Header().Set("Connection", "close")
		}
		w.WriteHeader(StatusBadRequest)
		return
	}
	var h Handler
	var pat *pattern
	var matches []string
	h, r.Pattern, pat, matches = mux.findHandler(r.Method, r.Host, r.URL.EscapedPath(), r.URL.RawQuery)
	setPathValues(r, pat, matches)
	h.ServeHTTP(w, r)
}

func setPathValues(r *Request, pat *pattern, matches []string) {
	if pat == nil {
		return
	}
	i := 0
	for _, seg := range pat.segments {
		if seg.wild && seg.s != "" {
			if i < len(matches) {
				r.SetPathValue(seg.s, matches[i])
			}
			i++
		}
	}
}

// Handle registers the handler for the given pattern.
func (mux *ServeMux) Handle(pattern string, handler Handler) {
	mux.register(pattern, handler)
}

// HandleFunc registers the handler function for the given pattern.
func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	mux.register(pattern, HandlerFunc(handler))
}

// Handle registers the handler for the given pattern in [DefaultServeMux].
func Handle(pattern string, handler Handler) {
	DefaultServeMux.register(pattern, handler)
}

// HandleFunc registers the handler function for the given pattern in [DefaultServeMux].
func HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	DefaultServeMux.register(pattern, HandlerFunc(handler))
}

func (mux *ServeMux) register(pattern string, handler Handler) {
	if err := mux.registerErr(pattern, handler); err != nil {
		panic(err)
	}
}

func (mux *ServeMux) registerErr(patstr string, handler Handler) error {
	if patstr == "" {
		return errors.New("http: invalid pattern")
	}
	if handler == nil {
		return errors.New("http: nil handler")
	}
	if f, ok := handler.(HandlerFunc); ok && f == nil {
		return errors.New("http: nil handler")
	}

	pat, err := parsePattern(patstr)
	if err != nil {
		return fmt.Errorf("parsing %q: %w", patstr, err)
	}

	_, file, line, ok := runtime.Caller(3)
	if !ok {
		pat.loc = "unknown location"
	} else {
		pat.loc = fmt.Sprintf("%s:%d", file, line)
	}

	mux.mu.Lock()
	defer mux.mu.Unlock()
	if err := mux.index.possiblyConflictingPatterns(pat, func(pat2 *pattern) error {
		if pat.conflictsWith(pat2) {
			d := describeConflict(pat, pat2)
			return fmt.Errorf("pattern %q (registered at %s) conflicts with pattern %q (registered at %s):\n%s",
				pat, pat.loc, pat2, pat2.loc, d)
		}
		return nil
	}); err != nil {
		return err
	}
	mux.tree.addPattern(pat, handler)
	mux.index.addPattern(pat)
	return nil
}

func validMethod(method string) bool {
	return httpguts.ValidHeaderFieldName(method)
}
