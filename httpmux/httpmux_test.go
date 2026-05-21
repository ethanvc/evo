package httpmux

import (
	"net/http/httptest"
	"slices"
	"testing"
)

func TestServeMuxTrimsTrailingSlashBeforeMatch(t *testing.T) {
	mux := NewServeMux()
	mux.HandleFunc("/api/", func(w ResponseWriter, _ *Request) {
		w.WriteHeader(204)
	})

	for _, path := range []string{"/api", "/api/"} {
		req := httptest.NewRequest("GET", "http://example.com"+path, nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != 204 {
			t.Fatalf("%s: status got %d, want 204", path, rec.Code)
		}
	}
}

func TestServeMuxFindHandlerReturnsNilWhenNoPatternMatches(t *testing.T) {
	mux := NewServeMux()
	mux.HandleFunc("GET /api", func(ResponseWriter, *Request) {})

	for _, test := range []struct {
		name, method, path string
	}{
		{name: "not found", method: "GET", path: "/missing"},
		{name: "method mismatch", method: "POST", path: "/api"},
	} {
		t.Run(test.name, func(t *testing.T) {
			n, matches := mux.Match(test.method, "", test.path)
			if n != nil {
				t.Fatalf("node got %v, want nil", n)
			}
			if matches != nil {
				t.Fatalf("matches got %v, want nil", matches)
			}
		})
	}
}

func TestServeMuxFindHandler(t *testing.T) {
	mux := NewServeMux()
	for _, pattern := range []string{
		"/item/",
		"POST /item/{user}",
		"GET /item/{user}",
		"/item/{user}",
		"/item/{user}/{id}",
		"/item/{user}/new",
		"/item/{$}",
		"POST alt.com/item/{user}",
		"GET /headwins",
		"HEAD /headwins",
		"/path/{p...}",
	} {
		mux.HandleFunc(pattern, func(ResponseWriter, *Request) {})
	}

	for _, test := range []struct {
		method, host, path string
		wantPattern        string
		wantMatches        []string
	}{
		{"GET", "", "/item/jba", "GET /item/{user}", []string{"jba"}},
		{"POST", "", "/item/jba", "POST /item/{user}", []string{"jba"}},
		{"HEAD", "", "/item/jba", "GET /item/{user}", []string{"jba"}},
		{"get", "", "/item/jba", "/item/{user}", []string{"jba"}},
		{"POST", "", "/item/jba/17", "/item/{user}/{id}", []string{"jba", "17"}},
		{"GET", "", "/item/jba/new", "/item/{user}/new", []string{"jba"}},
		{"GET", "", "/item/", "/item/", nil},
		{"GET", "", "/item/jba/17/line2", "", nil},
		{"POST", "alt.com", "/item/jba", "POST alt.com/item/{user}", []string{"jba"}},
		{"GET", "alt.com", "/item/jba", "GET /item/{user}", []string{"jba"}},
		{"GET", "", "/item", "/item/", nil},
		{"GET", "", "/headwins", "GET /headwins", nil},
		{"HEAD", "", "/headwins", "HEAD /headwins", nil},
		{"GET", "", "/path/to/file", "/path/{p...}", []string{"to/file"}},
		{"GET", "", "/path/*", "/path/{p...}", []string{"*"}},
	} {
		gotNode, gotMatches := mux.Match(test.method, test.host, test.path)
		if got := nodePatternString(gotNode); got != test.wantPattern {
			t.Errorf("%s %s%s: pattern got %q, want %q", test.method, test.host, test.path, got, test.wantPattern)
		}
		if !slices.Equal(gotMatches, test.wantMatches) {
			t.Errorf("%s %s%s: matches got %v, want %v", test.method, test.host, test.path, gotMatches, test.wantMatches)
		}
	}
}

func TestServeMuxFindHandlerWildcardEndings(t *testing.T) {
	for _, test := range []struct {
		name     string
		patterns []string
		checks   []matchPatternCheck
	}{
		{
			name:     "dollar matches only trailing slash",
			patterns: []string{"/a/b/{$}"},
			checks: []matchPatternCheck{
				{path: "/a/b", wantPattern: "", wantMatches: nil},
				{path: "/a/b/", wantPattern: "", wantMatches: nil},
				{path: "/a/b/c", wantPattern: "", wantMatches: nil},
				{path: "/a/b/c/d", wantPattern: "", wantMatches: nil},
			},
		},
		{
			name:     "single wildcard does not match trailing slash",
			patterns: []string{"/a/b/{w}"},
			checks: []matchPatternCheck{
				{path: "/a/b", wantPattern: "", wantMatches: nil},
				{path: "/a/b/", wantPattern: "", wantMatches: nil},
				{path: "/a/b/c", wantPattern: "/a/b/{w}", wantMatches: []string{"c"}},
				{path: "/a/b/c/d", wantPattern: "", wantMatches: nil},
			},
		},
		{
			name:     "multi wildcard matches remaining path",
			patterns: []string{"/a/b/{w...}"},
			checks: []matchPatternCheck{
				{path: "/a/b", wantPattern: "", wantMatches: nil},
				{path: "/a/b/", wantPattern: "", wantMatches: nil},
				{path: "/a/b/c", wantPattern: "/a/b/{w...}", wantMatches: []string{"c"}},
				{path: "/a/b/c/d", wantPattern: "/a/b/{w...}", wantMatches: []string{"c/d"}},
			},
		},
		{
			name:     "specific ending wins",
			patterns: []string{"/a/b/{$}", "/a/b/{w}", "/a/b/{w...}"},
			checks: []matchPatternCheck{
				{path: "/a/b", wantPattern: "", wantMatches: nil},
				{path: "/a/b/", wantPattern: "", wantMatches: nil},
				{path: "/a/b/c", wantPattern: "/a/b/{w}", wantMatches: []string{"c"}},
				{path: "/a/b/c/d", wantPattern: "/a/b/{w...}", wantMatches: []string{"c/d"}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			mux := NewServeMux()
			for _, pattern := range test.patterns {
				mux.HandleFunc(pattern, func(ResponseWriter, *Request) {})
			}
			for _, check := range test.checks {
				gotNode, gotMatches := mux.Match("GET", "", check.path)
				if got := nodePatternString(gotNode); got != check.wantPattern {
					t.Errorf("%s: pattern got %q, want %q", check.path, got, check.wantPattern)
				}
				if !slices.Equal(gotMatches, check.wantMatches) {
					t.Errorf("%s: matches got %v, want %v", check.path, gotMatches, check.wantMatches)
				}
			}
		})
	}
}

func TestServeMuxFindHandlerEscapedWildcards(t *testing.T) {
	for _, test := range []struct {
		pattern     string
		path        string
		wantMatches []string
	}{
		{
			pattern:     "/{a}/is/{b}/{c...}",
			path:        "/now/is/the/time/for/all",
			wantMatches: []string{"now", "the", "time/for/all"},
		},
		{
			pattern:     "/names/{name}/{other...}",
			path:        "/names/%2fjohn/address",
			wantMatches: []string{"/john", "address"},
		},
		{
			pattern:     "/names/{name}/{other...}",
			path:        "/names/john%2Fdoe/there/is%2F/more",
			wantMatches: []string{"john/doe", "there/is//more"},
		},
		{
			pattern:     "/names/{name}/{other...}",
			path:        "/names/n/*",
			wantMatches: []string{"n", "*"},
		},
	} {
		mux := NewServeMux()
		mux.HandleFunc(test.pattern, func(ResponseWriter, *Request) {})

		gotNode, gotMatches := mux.Match("GET", "", test.path)
		if got := nodePatternString(gotNode); got != test.pattern {
			t.Errorf("%s: pattern got %q, want %q", test.path, got, test.pattern)
		}
		if !slices.Equal(gotMatches, test.wantMatches) {
			t.Errorf("%s: matches got %v, want %v", test.path, gotMatches, test.wantMatches)
		}
	}
}

type matchPatternCheck struct {
	path        string
	wantPattern string
	wantMatches []string
}

func nodePatternString(n *routingNode) string {
	if n == nil {
		return ""
	}
	return n.pattern.String()
}
