package httpsvr

import (
	"fmt"

	"github.com/ethanvc/evo/httpsvr/ginradix"
)

type Router struct {
	raidx ginradix.Tree[*RouteNode]
}

func (r *Router) Register(pattern string, h *Handler, methodSlice ...string) {
	for _, method := range methodSlice {
		h, _, _ := r.Get(method, pattern)
		if h != nil {
			panic(fmt.Errorf("%s %s already exist, the formal func is %s", method, pattern, h.NameOfFunc()))
		}
	}
	routeNode := &RouteNode{}
	r.raidx.MustInsert(pattern, routeNode)
	routeNode.nodes = append(routeNode.nodes, Node{
		Handler: h,
		Method:  methodSlice,
	})
}

func (r *Router) Get(method string, pattern string) (*Handler, string, ginradix.Params) {
	n, params := r.raidx.Search(pattern, nil)
	if n == nil {
		return nil, "", nil
	}

	for _, nn := range n.Val.nodes {
		for _, m := range nn.Method {
			if m == method {
				return nn.Handler, n.Pattern, params
			}
		}
	}
	return nil, "", nil
}

type RouteNode struct {
	nodes []Node
}

type Node struct {
	Method  []string
	Handler *Handler
}
