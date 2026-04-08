package ginradix

import (
	"errors"
	"fmt"
	"strings"
)

type Node[Value any] struct {
	Part      string
	Pattern   string
	Children  []*Node[Value]
	WildChild *Node[Value]
	Val       Value
	ValValid  bool
}

func newNode[Value any](part string) *Node[Value] {
	return &Node[Value]{
		Part: part,
	}
}

func (n *Node[Value]) reset(part string) {
	n.Part = part
	n.Pattern = ""
	n.Children = nil
	n.WildChild = nil
	var defaultVal Value
	n.Val = defaultVal
	n.ValValid = false
}

func (n *Node[Value]) insert(pattern, restPattern string, val Value) error {
	if n.isWildNode() {
		return n.insertWild(pattern, restPattern, val)
	}
	return n.insertPlain(pattern, restPattern, val)
}

// n is a plain node
func (n *Node[Value]) insertPlain(pattern, restPattern string, val Value) error {
	if patternStartWithWild(restPattern) {
		// a vs :id
		return fmt.Errorf("ShouldNeverComeHere; pattern=%s, n.part=%s", pattern, n.Part)
	}
	currentPart := getNextPatternPart(restPattern)
	prefix := longestCommonPrefix(n.Part, currentPart)
	prefixLen := len(prefix)
	restPattern = restPattern[prefixLen:]
	if prefixLen == len(currentPart) && prefixLen == len(n.Part) {
		// a vs a
		if restPattern == "" {
			if n.ValValid {
				return fmt.Errorf("PatternAlreadyExist;pattern=%s", pattern)
			}
			n.setVal(pattern, val)
			return nil
		}
		candidate := n.getCandidate(restPattern)
		if candidate == nil {
			head, err := createNewNodes(pattern, restPattern, val)
			if err != nil {
				return err
			}
			n.insertChild(head)
			return nil
		}
		return candidate.insert(pattern, restPattern, val)
	}
	if prefixLen == len(n.Part) {
		// a vs ab
		candidate := n.getCandidate(restPattern)
		if candidate != nil {
			return candidate.insertPlain(pattern, restPattern, val)
		}
		newChildren, err := createNewNodes(pattern, restPattern, val)
		if err != nil {
			return err
		}
		n.insertChild(newChildren)
		return nil
	}
	if prefixLen == len(currentPart) {
		// ab vs a
		var newChild *Node[Value]
		if restPattern != "" {
			var err error
			newChild, err = createNewNodes(pattern, restPattern, val)
			if err != nil {
				return err
			}
		}
		oldChild := *n
		n.reset(prefix)
		oldChild.Part = oldChild.Part[prefixLen:]
		n.insertChild(&oldChild)
		if newChild == nil {
			n.setVal(pattern, val)
		} else {
			n.insertChild(newChild)
		}
		return nil
	}
	// ab vs ac
	newChildren, err := createNewNodes(pattern, restPattern, val)
	if err != nil {
		return err
	}
	oldChild := *n
	oldChild.Part = oldChild.Part[prefixLen:]
	n.reset(prefix)
	n.insertChild(&oldChild)
	n.insertChild(newChildren)
	return nil
}

func patternStartWithWild(pattern string) bool {
	return pattern[0] == ':' || pattern[0] == '*'
}

// n is a wild node
func (n *Node[Value]) insertWild(pattern, restPattern string, val Value) error {
	part := getNextPatternPart(restPattern)
	if !patternStartWithWild(part) {
		// :id vs a
		return fmt.Errorf("ShouldNeverHere: pattern=%s, n.part=%s", pattern, n.Part)
	}
	if part != n.Part {
		// :id vs :idx
		return fmt.Errorf("PatternConflict: pattern=%s, n.part=%s", pattern, n.Part)
	}
	// :id vs :id
	restPattern = restPattern[len(part):]
	if restPattern == "" {
		if n.ValValid {
			return fmt.Errorf("PatternAlreadyExist; pattern=%s", pattern)
		}
		n.setVal(pattern, val)
		return nil
	}
	candidate := n.getCandidate(restPattern)
	if candidate != nil {
		return candidate.insert(pattern, restPattern, val)
	}
	head, err := createNewNodes(pattern, restPattern, val)
	if err != nil {
		return err
	}
	n.insertChild(head)
	return nil
}

func (n *Node[Value]) getCandidate(pattern string) *Node[Value] {
	ch := pattern[0]
	if ch == ':' || ch == '*' {
		return n.WildChild
	}
	for _, child := range n.Children {
		if child.Part[0] == ch {
			return child
		}
	}
	return nil
}

func (n *Node[Value]) getPlainCandidate(pattern string) *Node[Value] {
	ch := pattern[0]
	for _, child := range n.Children {
		if child.Part[0] == ch {
			return child
		}
	}
	return nil
}

func (n *Node[Value]) setVal(pattern string, val Value) {
	n.Pattern = pattern
	n.Val = val
	n.ValValid = true
}

func (n *Node[Value]) isWildNode() bool {
	return n.Part[0] == ':' || n.Part[0] == '*'
}

func (n *Node[Value]) insertChild(child *Node[Value]) {
	if child.isWildNode() {
		n.WildChild = child
	} else {
		n.Children = append(n.Children, child)
	}
}

func (n *Node[Value]) consume(p string) int {
	firstByte := n.Part[0]
	if firstByte == ':' {
		for i, ch := range p {
			if ch == '/' {
				return i
			}
		}
		return len(p)
	}
	if firstByte == '*' {
		return len(p)
	}
	if strings.HasPrefix(p, n.Part) {
		return len(n.Part)
	}
	return 0
}

type Tree[Value any] struct {
	root *Node[Value]
}

func (t *Tree[Value]) MustInsert(pattern string, val Value) {
	err := t.Insert(pattern, val)
	if err != nil {
		panic(err)
	}
}
func (t *Tree[Value]) Insert(pattern string, val Value) error {
	if pattern == "" {
		return errors.New("empty pattern")
	}
	if pattern[0] != '/' {
		return errors.New("pattern must start with '/'")
	}
	var err error
	if t.root == nil {
		t.root, err = createNewNodes(pattern, pattern, val)
		return err
	}
	return t.root.insert(pattern, pattern, val)
}

type searchNode[Value any] struct {
	n        *Node[Value]
	params   Params
	restPath string
}

type searchNodes[Value any] []searchNode[Value]

func (nodes *searchNodes[Value]) empty() bool {
	if nodes == nil {
		return true
	}
	return len(*nodes) == 0
}

func (nodes *searchNodes[Value]) pop() (*Node[Value], Params, string) {
	last := (*nodes)[len(*nodes)-1]
	*nodes = (*nodes)[:len(*nodes)-1]
	return last.n, last.params, last.restPath
}

func (nodes *searchNodes[Value]) push(n *Node[Value], params Params, restPath string) {
	*nodes = append(*nodes, searchNode[Value]{n, params, restPath})
}

func (t *Tree[Value]) Search(p string, params Params) (*Node[Value], Params) {
	if t.root == nil || p == "" {
		return nil, params
	}

	n := t.root
	restPath := p
	var backNodes searchNodes[Value]
	for {
		if n == nil {
			if backNodes.empty() {
				return nil, params
			}
			n, params, restPath = backNodes.pop()
			continue
		}
		consumed := n.consume(restPath)
		if consumed == 0 {
			n = nil
			continue
		}
		// current node matched
		if n.isWildNode() {
			params = append(params, Param{
				Key:   n.Part[1:],
				Value: restPath[:consumed],
			})
		}
		if consumed == len(restPath) && n.ValValid {
			// fully matched
			return n, params
		}
		// match current node, but still need child match
		restPath = restPath[consumed:]
		candidate := n.getPlainCandidate(restPath)
		if candidate == nil {
			candidate = n.WildChild
		} else {
			if n.WildChild != nil {
				backNodes.push(n.WildChild, params, restPath)
			}
		}
		n = candidate
	}
}

func createNewNodes[Value any](pattern string, restPattern string, val Value) (*Node[Value], error) {
	var head *Node[Value]
	var tail *Node[Value]
	for restPattern != "" {
		part := getNextPatternPart(restPattern)
		tmp := newNode[Value](part)
		if head == nil {
			head = tmp
		}
		if tail != nil {
			tail.insertChild(tmp)
		}
		tail = tmp
		restPattern = restPattern[len(part):]
	}
	tail.setVal(pattern, val)
	return head, nil
}

func getNextPatternPart(pattern string) string {
	firstByte := pattern[0]
	if firstByte == ':' || firstByte == '*' {
		idx := strings.IndexByte(pattern, '/')
		if idx != -1 {
			return pattern[:idx]
		}
		return pattern
	}

	slashFound := false
	for i, ch := range pattern {
		if slashFound {
			if ch == ':' || ch == '*' {
				return pattern[:i]
			}
		}
		if ch == '/' {
			slashFound = true
		} else {
			slashFound = false
		}
	}
	return pattern
}

type Param struct {
	Key   string
	Value string
}

type Params []Param
