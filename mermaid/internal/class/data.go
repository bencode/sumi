package class

import "github.com/bencode/sumi/mermaid/internal/syntax"

// A Kind is what a Node is.
type Kind uint8

const (
	Class Kind = iota
	Namespace
	Note
	Interface // the invisible node at the far end of a lollipop
)

// A Member is an attribute or a method of a class.
type Member struct {
	Visibility, ID, Parameters, ReturnType, Classifier string
	Text                                               string // what the box shows, as Markdown, with HTML's &lt; and &gt;
}

// A Node is a namespace, a class, a note or an interface.
type Node struct {
	ID, Parent  string
	Kind        Kind
	Label       string // a class's text with its type, a note's text, a namespace's label, an interface's
	Annotations []string
	Members     []Member // a class's attributes
	Methods     []Member
	Classes     string // cssClasses, in one string as defineClass looks inside it
	Styles      []string
	At          syntax.Pos
}

// An Edge is a relation, or the line from a note to its class.
type Edge struct {
	ID, Start, End, Label  string
	StartLabel, EndLabel   string // the "1" and "many" at its ends
	StartMarker, EndMarker string // aggregation, extension, composition, dependency, lollipop or none
	Pattern                string // solid, dashed or dotted
	At                     syntax.Pos
}

// A Diagram is what getData gives, sizes aside.
type Diagram struct {
	Direction string
	Nodes     []Node
	Edges     []Edge
}

// Read reads a class diagram made ready by syntax.Prepare.
func Read(body string) (*Diagram, error) {
	ts, err := Lex(body)
	if err != nil {
		return nil, err
	}
	stmts, err := Parse(ts)
	if err != nil {
		return nil, err
	}
	return collect(stmts)
}
