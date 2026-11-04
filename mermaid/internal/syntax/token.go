// Package syntax holds what the Mermaid diagram parsers share: tokens and
// their places, scanning helpers, the statements every diagram accepts, the
// preparation Mermaid gives a diagram's source before parsing it, and errors.
package syntax

// A Pos is a place in a diagram's source: a line and a column, both from 1,
// with the column counted in characters.
type Pos struct{ Line, Col int }

// A Kind is what a token is. A diagram defines its own kinds from FirstKind.
type Kind int

const (
	EOF      Kind = iota
	Newline       // a line break or a ;
	Title         // its Text is the title
	AccTitle      // its Text is the accessible title
	AccDescr      // its Text is the accessible description
	FirstKind
)

// A Token is a piece of source: its kind, its text and where it starts.
type Token struct {
	Kind Kind
	Text string
	Pos  Pos
}
