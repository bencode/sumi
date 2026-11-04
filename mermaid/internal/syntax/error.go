package syntax

import "fmt"

// An Error is a mistake in a diagram's source, at the place it was found.
type Error struct {
	Line, Col   int
	Msg         string
	unsupported bool
}

func (e *Error) Error() string { return fmt.Sprintf("line %d:%d: %s", e.Line, e.Col, e.Msg) }

// Errorf is an Error at p.
func Errorf(p Pos, format string, args ...any) *Error {
	return &Error{Line: p.Line, Col: p.Col, Msg: fmt.Sprintf(format, args...)}
}

// Unsupported is the error for syntax Mermaid has that sumi does not yet.
func Unsupported(p Pos, what string) *Error {
	e := Errorf(p, "%s is not supported yet", what)
	e.unsupported = true
	return e
}

// IsUnsupported is whether e is an Unsupported error.
func (e *Error) IsUnsupported() bool { return e.unsupported }
