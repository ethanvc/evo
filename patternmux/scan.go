package patternmux

import (
	"strings"
	"sync"
)

// scanEntry is a registered pattern handled by the scan backend.
type scanEntry[T any] struct {
	node     *Node[T]
	segments []Segment
}

// convertedBufPool reuses byte buffers for Converted assembly when HasKeep.
var convertedBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 64)
		return &b
	},
}

func getConvertedBuf() *[]byte {
	bp := convertedBufPool.Get().(*[]byte)
	*bp = (*bp)[:0]
	return bp
}

func putConvertedBuf(bp *[]byte) {
	if bp == nil {
		return
	}
	*bp = (*bp)[:0]
	convertedBufPool.Put(bp)
}

// tryScan walks segments against input. When caps != nil and writeConverted == true,
// captures and converted bytes are emitted into the provided buffers.
// Returns true if segments fully consume input.
func tryScan(segments []Segment, input string, caps *Captures, buf *[]byte) bool {
	pos := 0
	for _, seg := range segments {
		switch s := seg.(type) {
		case Literal:
			if pos+len(s.Text) > len(input) || input[pos:pos+len(s.Text)] != s.Text {
				return false
			}
			if buf != nil {
				*buf = append(*buf, s.Text...)
			}
			pos += len(s.Text)
		case Expr:
			end, ok := consumeByRules(s.Rules, input, pos)
			if !ok {
				return false
			}
			sub := input[pos:end]
			if caps != nil {
				*caps = append(*caps, Capture{Key: s.Name, Value: sub})
			}
			if buf != nil && s.Action == ActionKeep {
				*buf = append(*buf, sub...)
			}
			pos = end
		}
	}
	return pos == len(input)
}

// consumeByRules picks the end position for one expression by intersecting all
// rule-derived boundaries. The consumed substring [pos, end) is guaranteed to
// satisfy every rule's character class because each character-class rule's own
// boundary lies exactly at the first violating char.
//
// Consumption length must be > 0; expressions cannot match an empty substring.
func consumeByRules(rules []Rule, input string, pos int) (end int, ok bool) {
	end = len(input)
	for _, r := range rules {
		boundary := ruleBoundary(r, input, pos)
		if boundary < end {
			end = boundary
		}
	}
	if end <= pos {
		return 0, false
	}
	return end, true
}

func ruleBoundary(r Rule, input string, pos int) int {
	switch r {
	case RuleUntilSlash:
		if i := strings.IndexByte(input[pos:], '/'); i >= 0 {
			return pos + i
		}
		return len(input)
	case RuleUntilBlank:
		for i := pos; i < len(input); i++ {
			if isBlank(input[i]) {
				return i
			}
		}
		return len(input)
	case RuleRest:
		return len(input)
	case RuleDigit:
		i := pos
		for i < len(input) && isDigit(input[i]) {
			i++
		}
		return i
	case RuleHexDigit:
		i := pos
		for i < len(input) && isHexDigit(input[i]) {
			i++
		}
		return i
	}
	return len(input)
}

func isBlank(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') ||
		(b >= 'a' && b <= 'f') ||
		(b >= 'A' && b <= 'F')
}
