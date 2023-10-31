package evohttp

import (
	"regexp"
	"strings"
)

type Router struct {
	methodNodes []*MethodRouteNode
}

func NewRouter() *Router {
	r := &Router{}
	return r
}

func (r *Router) Find(httpMethod, urlPath string, params map[string]string) *RouteNode {
	var mn *MethodRouteNode
	for _, mn = range r.methodNodes {
		if mn.HttpMethod == httpMethod {
			break
		}
	}
	if mn == nil {
		return nil
	}
	n := mn.Node
	p := urlPath
	for n != nil {
		if n.isParamNode() {
			if n.pathPart[0] == '*' {
				params[n.pathPart[1:]] = p
				p = ""
			} else {
				slashIndex := strings.IndexRune(p, '/')
				if slashIndex == -1 {
					params[n.pathPart[1:]] = p
					p = ""
				} else {
					params[n.pathPart[1:]] = p[0:slashIndex]
					p = p[slashIndex:]
				}
			}
		} else {
			if !strings.HasPrefix(p, n.pathPart) {
				return nil
			}
			p = p[len(n.pathPart):]
		}
		if len(p) == 0 {
			if n.Handlers != nil {
				return n
			} else {
				// for abc.com/abc/*filepath, we want a match and filepath = ""
				if len(n.children) == 1 && n.children[0].isParamNode() {
					return n.children[0]
				}
				return nil
			}
		}
		oldN := n
		n = nil
		for _, child := range oldN.children {
			if child.isParamNode() || child.pathPart[0] == p[0] {
				n = child
				break
			}
		}
	}
	return nil
}

func (r *Router) ListAll() []*MethodRouteNode {
	var result []*MethodRouteNode
	for _, n := range r.methodNodes {
		n.Node.visit(func(nn *RouteNode) bool {
			if len(nn.Handlers) > 0 {
				result = append(result, &MethodRouteNode{
					HttpMethod: n.HttpMethod,
					Node:       nn,
				})
			}
			return true
		})
	}
	return result
}

func (r *Router) addRoute(httpMethod, urlPath string, handlers HandlerChain) {
	assert(len(httpMethod) > 0, "HttpMethod invalid")
	assert(urlPath[0] == '/', "urlPathInvalid")
	assert(len(handlers) > 0, "handlers empty")
	node := r.mustGetRootNode(httpMethod)
	node.addRoute(urlPath, handlers)
}

func (r *Router) mustGetRootNode(httpMethod string) *RouteNode {
	for _, n := range r.methodNodes {
		if n.HttpMethod == httpMethod {
			return n.Node
		}
	}
	n := &MethodRouteNode{
		HttpMethod: httpMethod,
		Node:       newHandlerNode(),
	}
	r.methodNodes = append(r.methodNodes, n)
	return n.Node
}

type RouteNode struct {
	pathPart string
	Handlers HandlerChain
	children []*RouteNode
	FullPath string
}

func newHandlerNode() *RouteNode {
	n := &RouteNode{}
	return n
}

func (n *RouteNode) isParamNode() bool {
	if len(n.pathPart) == 0 {
		return false
	}
	return n.isWildChar(n.pathPart[0])
}

func (n *RouteNode) visit(f func(n *RouteNode) bool) bool {
	if ok := f(n); !ok {
		return false
	}
	for _, child := range n.children {
		if ok := child.visit(f); !ok {
			return false
		}
	}
	return true
}

func (n *RouteNode) addRoute(urlPath string, handlers HandlerChain) {
	parts := splitPatternPath(urlPath)
	n.addRouteInternal(urlPath, parts, handlers)
}

func (n *RouteNode) addRouteInternal(fullPath string, parts []string, handlers HandlerChain) {
	if len(n.pathPart) == 0 {
		n.insertNodes(fullPath, parts, handlers)
		return
	}
	if len(parts) == 0 {
		panic("why here")
	}
	part := parts[0]
	if part == n.pathPart {
		child := n.findIntersectionChild(parts[1])
		if child == nil {
			child = &RouteNode{}
			n.children = append(n.children, child)
		}
		child.addRouteInternal(fullPath, parts[1:], handlers)
		return
	}
	if n.isParamNode() || n.isWildChar(part[0]) {
		panic("conflict")
	}
	prefixLen := getCommonPrefixLength(n.pathPart, part)
	if prefixLen == len(n.pathPart) {
		parts = append([]string{part[prefixLen:]}, parts[1:]...)
		for _, child := range n.children {
			if child.isParamNode() {
				panic("conflict with param node")
			}
			if getCommonPrefixLength(child.pathPart, parts[0]) > 0 {
				child.addRouteInternal(fullPath, parts, handlers)
				return
			}
		}
		newNode := &RouteNode{}
		n.children = append(n.children, newNode)
		newNode.insertNodes(fullPath, append([]string{part[prefixLen:]}, parts[1:]...), handlers)
		return
	}
	newRoot := *n
	newRoot.pathPart = n.pathPart[prefixLen:]
	n.pathPart = part[0:prefixLen]
	n.Handlers = nil
	n.children = nil
	n.FullPath = ""
	newNode := &RouteNode{}
	n.children = append(n.children, &newRoot, newNode)
	newNode.insertNodes(fullPath, append([]string{part[prefixLen:]}, parts[1:]...), handlers)
}

func getCommonPrefixLength(s1, s2 string) int {
	l := len(s1)
	if l > len(s2) {
		l = len(s2)
	}
	for i := 0; i < l; i++ {
		if s1[i] == s2[i] {
			continue
		}
		return i
	}
	return l
}

func (n *RouteNode) findIntersectionChild(part string) *RouteNode {
	for _, child := range n.children {
		if child.pathPart[0] == part[0] {
			return child
		}
	}
	return nil
}

func (n *RouteNode) insertNodes(fullPath string, parts []string, handlers HandlerChain) {
	if len(parts) == 1 {
		n.pathPart = parts[0]
		n.Handlers = handlers
		n.FullPath = fullPath
		return
	}
	n.pathPart = parts[0]
	newNode := &RouteNode{}
	n.children = append(n.children, newNode)
	newNode.insertNodes(fullPath, parts[1:], handlers)
}

func (n *RouteNode) isWildChar(ch byte) bool {
	return ch == '*' || ch == ':'
}

type MethodRouteNode struct {
	HttpMethod string
	Node       *RouteNode
}

func splitPatternPath(patternPath string) []string {
	var result []string
	for {
		m := regexWildPart.FindStringIndex(patternPath)
		if m == nil {
			if len(patternPath) > 0 {
				result = append(result, patternPath)
			}
			break
		}
		result = append(result, patternPath[0:m[0]+1])
		if patternPath[m[1]-1] == '/' {
			result = append(result, patternPath[m[0]+1:m[1]-1])
			patternPath = patternPath[m[1]-1:]
		} else {
			result = append(result, patternPath[m[0]+1:m[1]])
			patternPath = patternPath[m[1]:]
		}
	}
	return result
}

var regexWildPart = regexp.MustCompile(`/([:*][a-zA-Z0-9-_]+)/?`)
