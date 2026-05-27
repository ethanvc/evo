package httpsvr

import (
	"fmt"

	"github.com/ethanvc/evo/httpmux"
)

type Router struct {
	mux *httpmux.HttpMux[*Handler]
}

func (r *Router) initMux() {
	if r.mux == nil {
		r.mux = httpmux.New[*Handler]()
	}
}

func (r *Router) Register(pattern string, h *Handler, methodSlice ...string) {
	r.initMux()
	for _, method := range methodSlice {
		h, _, _ := r.Get(method, pattern)
		if h != nil {
			panic(fmt.Errorf("%s %s already exist, the formal func is %s", method, pattern, h.NameOfFunc()))
		}
	}
	for _, method := range methodSlice {
		r.mux.Register(method, pattern, h)
	}
}

func (r *Router) Get(method string, path string) (*Handler, string, httpmux.Params) {
	r.initMux()
	n, ps, _ := r.mux.Lookup(method, path)
	if n == nil {
		if ps != nil {
			httpmux.PutParams(ps)
		}
		return nil, "", nil
	}
	var out httpmux.Params
	if ps != nil {
		out = append(httpmux.Params{}, *ps...)
		httpmux.PutParams(ps)
	}
	return n.Value(), n.Pattern(), out
}
