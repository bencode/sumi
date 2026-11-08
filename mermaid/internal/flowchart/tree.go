package flowchart

import "github.com/bencode/sumi/mermaid/internal/syntax"

// A Stmt is one thing a diagram says. Those in a list are in the order Mermaid's
// parser reduces them, which is the order they take effect.
type Stmt interface{ stmt() }

// A Chain is a line of nodes and links.
type Chain struct {
	Steps   []Stmt   // Vertex, Data, Class and Link
	Members []string // the statement's value: all its nodes
}

// A Vertex names a node (addVertex). With brackets it has Text and a Shape.
type Vertex struct {
	Pos   syntax.Pos
	ID    string
	Text  *Label
	Shape Shape
}

// Data is the @{...} on a node.
type Data struct {
	Pos  syntax.Pos
	ID   string
	Text string // as the lexer gives it
}

// A Link joins each of From to each of To (addLink).
type Link struct {
	Pos          syntax.Pos
	From, To     []string
	ID           string // "" unless the source gives one
	Text         *Label
	Type, Stroke string
}

// A Style is a style statement.
type Style struct {
	Pos    syntax.Pos
	ID     string
	Styles []string
}

// A LinkStyle is a linkStyle statement. It has no Styles when all it sets is the
// curve, which nothing reads (the package doc says why).
type LinkStyle struct {
	Pos       syntax.Pos
	Positions []syntax.Token // none for default
	Styles    []string
}

// A ClassDef is a classDef statement.
type ClassDef struct {
	Pos    syntax.Pos
	IDs    string
	Styles []string
}

// A Class adds a class to ids, split by commas: a class statement, a ::: or a click.
type Class struct{ IDs, Class string }

// A Subgraph is a subgraph ... end. ID and Title are the same Label when only one was
// written, which is what addSubGraph asks.
type Subgraph struct {
	Pos       syntax.Pos
	ID, Title *Label
	Body      []Stmt
}

// A Direction is a direction statement, which only a subgraph reads.
type Direction struct{ Value string }

// A Member is a statement whose value is text: accTitle and accDescr. A subgraph takes
// that value for the id of a member, as the grammar does.
type Member struct{ Text string }

func (*Chain) stmt()     {}
func (*Vertex) stmt()    {}
func (*Data) stmt()      {}
func (*Link) stmt()      {}
func (*Style) stmt()     {}
func (*LinkStyle) stmt() {}
func (*ClassDef) stmt()  {}
func (*Class) stmt()     {}
func (*Subgraph) stmt()  {}
func (*Direction) stmt() {}
func (*Member) stmt()    {}
