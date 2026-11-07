package flowchart

import (
	"slices"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// A Diagram is what getData gives, sizes aside.
type Diagram struct {
	Direction string // TB, BT, LR, RL or BR
	Nodes     []Node // subgraphs, then vertices
	Edges     []Edge
}

// A Shape is a shape the grammar can write.
type Shape uint8

const (
	Rect Shape = iota
	Round
	Stadium
	Subroutine
	Cylinder
	Circle
	DoubleCircle
	Diamond
	Hexagon
	Odd
	Trapezoid
	InvTrapezoid
	LeanRight
	LeanLeft
	ellipse // what (- -) writes, and Mermaid has no such shape to draw
)

// A Label is text and how to read it: "text", "string" or "markdown". Its entity codes
// are as syntax.Prepare left them.
type Label struct{ Text, Kind string }

// A Node is a vertex, or a subgraph if it is a Group.
type Node struct {
	ID, Parent string // Parent is empty at the root
	Label      Label
	Group      bool
	Shape      Shape    // not for a group
	Dir        string   // a group's direction, as written
	Styles     []string // cssStyles
	Compiled   []string // cssCompiledStyles
}

// An Edge is a link between two nodes.
type Edge struct {
	ID, Start, End         string
	Label                  Label
	Stroke                 string // normal, thick, dotted, invisible or INVALID
	StartMarker, EndMarker string // none, arrow_point, arrow_circle or arrow_cross
	Styles                 []string
	Compiled               []string
}

// data is the diagram the tables hold (getData), in the direction dir.
func (t *tables) data(dir string) (*Diagram, error) {
	parents := map[string]string{}
	for _, sg := range t.subgraphs {
		for _, id := range sg.nodes {
			parents[id] = sg.id
		}
	}
	d := &Diagram{Direction: dir}
	groups := map[string]int{}
	for i := len(t.subgraphs) - 1; i >= 0; i-- {
		sg := t.subgraphs[i]
		d.Nodes = append(d.Nodes, Node{
			ID: sg.id, Parent: parents[sg.id], Label: sg.label, Group: true, Dir: sg.dir,
			Compiled: t.compiled(sg.classes),
		})
	}
	for i, n := range d.Nodes {
		groups[n.ID] = i
	}
	for _, v := range t.vertices {
		if i, ok := groups[v.id]; ok {
			d.Nodes[i].Styles, d.Nodes[i].Compiled = v.styles, t.compiled(v.classes)
			continue
		}
		if v.shape == ellipse {
			return nil, syntax.Errorf(v.at, "Mermaid has no shape (- -)")
		}
		d.Nodes = append(d.Nodes, Node{
			ID: v.id, Parent: parents[v.id], Label: v.label, Shape: v.shape, Styles: v.styles,
			Compiled: t.compiled(append([]string{"default", "node"}, v.classes...)),
		})
	}
	for _, e := range t.edges {
		from, to := markers(e.kind)
		d.Edges = append(d.Edges, Edge{
			ID: e.id, Start: e.start, End: e.end, Label: e.label, Stroke: e.stroke,
			StartMarker: from, EndMarker: to,
			Styles:   append(append([]string{}, t.defaultStyle...), e.styles...),
			Compiled: t.compiled(e.classes),
		})
	}
	return d, nil
}

// compiled are the styles of classes (getCompiledStyles).
func (t *tables) compiled(classes []string) []string {
	var styles []string
	for _, name := range classes {
		c := t.classes[name]
		if c == nil {
			continue
		}
		for _, s := range slices.Concat(c.styles, c.textStyles) {
			styles = append(styles, trim(s))
		}
	}
	return styles
}
