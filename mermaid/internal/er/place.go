package er

import (
	"slices"

	"github.com/bencode/sumi/mermaid/internal/elk"
	"github.com/bencode/sumi/mermaid/internal/graphlayout"
	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// What getData and the labels' defaults give what Size does not measure.
const (
	edgeWrap     = 200 // createText's default
	titleWrap    = 120 // a Markdown title's, which measureGroupLabel still wraps as labelHelper does
	groupPadding = 8
)

// Placement is a diagram sized and laid out: what drawing reads.
type Placement struct {
	D      *Diagram
	Sized  []Sized     // by D.Nodes
	Titles []label.Box // by D.Nodes: a subgraph's
	Edges  []label.Box // by D.Edges
	R      graphlayout.Result
}

// Place sizes d and lays it out (defaultMeasureLayout, runElkLayoutCore).
func Place(d *Diagram, m scene.Measurer) (Placement, error) {
	if err := check(d); err != nil {
		return Placement{}, err
	}
	sized, err := Size(d, m)
	if err != nil {
		return Placement{}, err
	}
	p := Placement{D: d, Sized: sized, Titles: make([]label.Box, len(d.Nodes)), Edges: make([]label.Box, len(d.Edges))}
	data := graphlayout.Data{
		Direction: graphlayout.ElkDirection(d.Direction),
		Nodes:     make([]graphlayout.Node, len(d.Nodes)),
		Edges:     make([]graphlayout.Edge, len(d.Edges)),
	}
	for i, n := range d.Nodes {
		if data.Nodes[i], p.Titles[i], err = layoutNode(n, sized[i], m); err != nil {
			return Placement{}, err
		}
	}
	for i, e := range d.Edges {
		if data.Edges[i], p.Edges[i], err = layoutEdge(e, m); err != nil {
			return Placement{}, err
		}
	}
	if id := graphlayout.Crossing(data); id != "" {
		i := slices.IndexFunc(d.Edges, func(e Edge) bool { return e.ID == id })
		return Placement{}, syntax.Errorf(d.Edges[i].At, "a relationship between a subgraph and what it holds, under a subgraph with its own direction")
	}
	p.R = graphlayout.Layout(data)
	return p, nil
}

// check refuses two subgraphs with one id, which Mermaid hands ELK as they are, where
// layout needs a tree, and a subgraph with a blank name, whose empty id here means the
// root. A relationship written before the subgraph of its name ends at an entity
// getData then leaves out, and ELK's import throws.
func check(d *Diagram) error {
	nodes := make(map[string]bool, len(d.Nodes))
	for _, n := range d.Nodes {
		if nodes[n.ID] {
			return syntax.Unsupported(n.At, "two subgraphs with one name")
		}
		if n.ID == "" {
			return syntax.Unsupported(n.At, "a subgraph with a blank name")
		}
		nodes[n.ID] = true
	}
	for _, e := range d.Edges {
		for _, end := range []string{e.Start, e.End} {
			if !nodes[end] {
				return syntax.Errorf(e.At, "the relationship ends at %q, which a subgraph of that name replaced", end)
			}
		}
	}
	return nil
}

// layoutNode is what layout gets of n, and a subgraph's title.
func layoutNode(n Node, s Sized, m scene.Measurer) (graphlayout.Node, label.Box, error) {
	node := graphlayout.Node{ID: n.ID, Parent: n.Parent, Group: n.Group, Direction: elk.Undefined}
	if n.Dir != "" {
		node.Direction = graphlayout.ElkDirection(n.Dir)
	}
	if !n.Group {
		node.Width, node.Height = s.Shape.W, s.Shape.H
		return node, label.Box{}, nil
	}
	node.Padding = groupPadding
	if n.Label == "" {
		return node, label.Box{}, nil
	}
	lines, err := label.Read(n.Label, "markdown", n.At)
	if err != nil {
		return node, label.Box{}, err
	}
	title := label.Set(m, lines, titleWrap, 0)
	node.Title = sizeOf(title)
	return node, title, nil
}

// layoutEdge is what layout gets of e, and the text on it.
func layoutEdge(e Edge, m scene.Measurer) (graphlayout.Edge, label.Box, error) {
	edge := graphlayout.Edge{ID: e.ID, Start: e.Start, End: e.End} // its markers pull no line back
	if e.Label == "" {
		return edge, label.Box{}, nil
	}
	lines, err := label.ReadEdge(e.Label, "markdown", e.At)
	if err != nil {
		return edge, label.Box{}, err
	}
	text := label.Set(m, lines, edgeWrap, 0)
	edge.Label = sizeOf(text)
	return edge, text, nil
}

func sizeOf(b label.Box) *elk.Size { return &elk.Size{Width: b.W, Height: b.H} }
