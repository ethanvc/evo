package patternmux

import "errors"

var (
	ErrDuplicatePattern   = errors.New("patternmux: duplicate pattern")
	ErrDuplicateCanonical = errors.New("patternmux: duplicate canonical")
	ErrUnknownRule        = errors.New("patternmux: unknown rule")
	ErrMissingRule        = errors.New("patternmux: expression requires at least one rule")
	ErrInvalidSyntax      = errors.New("patternmux: invalid syntax")
	ErrEmptyPattern       = errors.New("patternmux: empty pattern")
)
