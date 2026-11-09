package er

import "github.com/bencode/sumi/mermaid/internal/syntax"

// An Attribute is a row of an entity.
type Attribute struct {
	Type, Name string
	Keys       []string
	Comment    string
}

// A Node is an entity or, when Group is set, a subgraph.
type Node struct {
	ID, Parent       string
	Label            string // an entity's name, a subgraph's title
	Alias            string
	Group            bool
	Attributes       []Attribute
	Classes          string // cssClasses, in one string
	Dir              string // a subgraph's own direction, or ""
	Compiled, Styles []string
	At               syntax.Pos
}

// An Edge is a relationship.
type Edge struct {
	ID, Start, End, Label  string
	StartMarker, EndMarker string // only_one, zero_or_one, one_or_more, zero_or_more or md_parent
	Pattern                string // solid or dashed
	At                     syntax.Pos
}

// A Diagram is what getData gives, sizes aside.
type Diagram struct {
	Direction string
	Nodes     []Node
	Edges     []Edge
}

// Read reads an entity relationship diagram made ready by syntax.Prepare.
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
