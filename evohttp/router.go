package evohttp

import (
	"regexp"
	"strings"
)

type Router struct {
	methodNodes []*methodNode
}

func NewRouter() *Router {
	r := &Router{}
	return r
}

func (r *Router) Find(httpMethod, urlPath string, params map[string]string) *handlerNode {
	var mn *methodNode
	for _, mn = range r.methodNodes {
		if mn.httpMethod == httpMethod {
			break
		}
	}
	if mn == nil {
		return nil
	}
	n := mn.node
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
			if n.handlers != nil {
				return n
			} else {
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

func (r *Router) ListAll() []*RouterItem {
	var result []*RouterItem
	for _, n := range r.methodNodes {
		n.node.visit(func(nn *handlerNode) bool {
			if len(nn.handlers) > 0 {
				result = append(result, &RouterItem{
					HttpMethod: n.httpMethod,
					Handlers:   nn.handlers,
					FullPath:   nn.fullPath,
				})
			}
			return true
		})
	}
	return result
}

func (r *Router) addRoute(httpMethod, urlPath string, handlers HandlerChain) {
	assert(len(httpMethod) > 0, "httpMethod invalid")
	assert(urlPath[0] == '/', "urlPathInvalid")
	assert(len(handlers) > 0, "handlers empty")
	node := r.mustGetRootNode(httpMethod)
	node.addRoute(urlPath, handlers)
}

func (r *Router) mustGetRootNode(httpMethod string) *handlerNode {
	for _, n := range r.methodNodes {
		if n.httpMethod == httpMethod {
			return n.node
		}
	}
	n := &methodNode{
		httpMethod: httpMethod,
		node:       newHandlerNode(),
	}
	r.methodNodes = append(r.methodNodes, n)
	return n.node
}

type handlerNode struct {
	pathPart string
	handlers HandlerChain
	children []*handlerNode
	fullPath string
}

func newHandlerNode() *handlerNode {
	n := &handlerNode{}
	return n
}

func (n *handlerNode) isParamNode() bool {
	if len(n.pathPart) == 0 {
		return false
	}
	return n.isWildChar(n.pathPart[0])
}

func (n *handlerNode) visit(f func(n *handlerNode) bool) bool {
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

func (n *handlerNode) addRoute(urlPath string, handlers HandlerChain) {
	n.addRouteInternal(urlPath, splitPatternPath(urlPath), handlers)
}

func (n *handlerNode) addRouteInternal(fullPath string, parts []string, handlers HandlerChain) {
	if len(n.pathPart) == 0 {
		n.insertNodes(fullPath, parts, handlers)
		return
	}
	if len(parts) == 1 {
		panic("why here")
	}
	part := parts[0]
	if part == n.pathPart {
		child := n.findIntersectionChild(parts[1])
		if child == nil {
			child = &handlerNode{}
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
		newNode := &handlerNode{}
		n.children = append(n.children, newNode)
		newNode.insertNodes(fullPath, append([]string{part[prefixLen:]}, parts[1:]...), handlers)
		return
	}
	newRoot := *n
	newRoot.pathPart = n.pathPart[prefixLen:]
	n.pathPart = part[0:prefixLen]
	n.handlers = nil
	n.children = nil
	n.fullPath = ""
	newNode := &handlerNode{}
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

func (n *handlerNode) findIntersectionChild(part string) *handlerNode {
	for _, child := range n.children {
		if child.pathPart[0] == part[0] {
			return child
		}
	}
	return nil
}

func (n *handlerNode) insertNodes(fullPath string, parts []string, handlers HandlerChain) {
	if len(parts) == 1 {
		n.pathPart = parts[0]
		n.handlers = handlers
		n.fullPath = fullPath
		return
	}
	n.pathPart = parts[0]
	newNode := &handlerNode{}
	n.children = append(n.children, newNode)
	newNode.insertNodes(fullPath, parts[1:], handlers)
}

func (n *handlerNode) isWildChar(ch byte) bool {
	return ch == '*' || ch == ':'
}

type methodNode struct {
	httpMethod string
	node       *handlerNode
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

type RouterItem struct {
	HttpMethod string
	FullPath   string
	Handlers   HandlerChain
}
