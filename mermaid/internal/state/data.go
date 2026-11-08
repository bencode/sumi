package state

import "github.com/bencode/sumi/mermaid/internal/syntax"

// A Shape is what a state is drawn as.
type Shape uint8

const (
	NoShape       Shape = iota // a state first named by a note
	Rect                       // rect
	RectWithTitle              // rectWithTitle
	Start                      // stateStart
	End                        // stateEnd
	Fork                       // fork
	Join                       // join
	Choice                     // choice
	Note                       // note
	Composite                  // roundedWithTitle
	Divider                    // divider
	NoteGroup                  // noteGroup
)

// A Node is a state, a note, or a group of them.
type Node struct {
	ID, Parent, Label string // Parent is empty at the root
	Descriptions      []string
	Shape             Shape
	Group             bool
	Dir               string // a group's direction
	Alt               bool   // statediagram-cluster-alt, every other level of groups
	Position          string // a note's "left of" or "right of"
	Styles, Compiled  []string
	At                syntax.Pos // where it was last named
}

// An Edge is a transition, or a note's line to its state.
type Edge struct {
	ID, Start, End, Label string
	Note                  bool
}

// A Diagram is what getData gives, sizes aside.
type Diagram struct {
	Direction string
	Nodes     []Node
	Edges     []Edge
}

// Read reads a state diagram made ready by syntax.Prepare.
func Read(body string) (*Diagram, error) {
	ts, err := Lex(body)
	if err != nil {
		return nil, err
	}
	stmts, err := Parse(ts)
	if err != nil {
		return nil, err
	}
	// Twice, as parsing and drawing each do: the id of `state "x" as [*] {` has a space
	// after it, so that it is only trimmed the first time, and its body named after it.
	doc := translateDoc("root", syntax.Pos{}, translateDoc("root", syntax.Pos{}, stmts))
	return fetch(doc, extract(doc))
}
