package class

import (
	"math"

	"github.com/bencode/sumi/mermaid/internal/elk"
	"github.com/bencode/sumi/mermaid/internal/graphlayout"
	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// Mermaid's defaults for what Size does not measure.
const (
	edgeWrap    = 200 // createText's default
	endTextSize = 11  // .edgeTerminals' font-size
)

// Placement is a diagram sized and laid out: what drawing reads.
type Placement struct {
	D            *Diagram
	Sized        []Sized      // by D.Nodes
	Titles       []label.Box  // by D.Nodes: a namespace's
	Edges        []label.Box  // by D.Edges
	Starts, Ends []*label.Box // by D.Edges: nil where it has none
	R            graphlayout.Result
}

// Place sizes d and lays it out (defaultMeasureLayout, runElkLayoutCore).
func Place(d *Diagram, m scene.Measurer) (Placement, error) {
	if err := uniqueIDs(d); err != nil {
		return Placement{}, err
	}
	sized, err := Size(d, m)
	if err != nil {
		return Placement{}, err
	}
	p := Placement{
		D: d, Sized: sized, Titles: make([]label.Box, len(d.Nodes)), Edges: make([]label.Box, len(d.Edges)),
		Starts: make([]*label.Box, len(d.Edges)), Ends: make([]*label.Box, len(d.Edges)),
	}
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
		if p.Starts[i], err = endText(e.StartLabel, e.At, m); err != nil {
			return Placement{}, err
		}
		if p.Ends[i], err = endText(e.EndLabel, e.At, m); err != nil {
			return Placement{}, err
		}
		data.Edges[i].StartText, data.Edges[i].EndText = sizeOf(p.Starts[i]), sizeOf(p.Ends[i])
	}
	p.R = graphlayout.Layout(data)
	return p, nil
}

// uniqueIDs refuses two nodes with one id, a class named as a namespace or as a
// generated note. Mermaid hands ELK both and paints what comes of it; here the
// nodes must form a tree.
func uniqueIDs(d *Diagram) error {
	seen := make(map[string]bool, len(d.Nodes))
	for _, n := range d.Nodes {
		if seen[n.ID] {
			return syntax.Unsupported(n.At, "a class and a namespace or note with one name")
		}
		seen[n.ID] = true
	}
	return nil
}

// layoutNode is what layout gets of n, and a namespace's title.
func layoutNode(n Node, s Sized, m scene.Measurer) (graphlayout.Node, label.Box, error) {
	node := graphlayout.Node{ID: n.ID, Parent: n.Parent, Direction: elk.Undefined}
	if n.Kind != Namespace {
		node.Width, node.Height, node.Intersect = s.Shape.W, s.Shape.H, graphlayout.ShapeIntersect(s.Shape)
		return node, label.Box{}, nil
	}
	node.Group = true
	lines, err := label.Read(n.Label, "string", n.At)
	if err != nil {
		return node, label.Box{}, err
	}
	title := label.Set(m, lines, math.Inf(1), 0)
	if title.W > 0 {
		node.Title = sizeOf(&title)
	}
	return node, title, nil
}

// layoutEdge is what layout gets of e, and the text on it.
func layoutEdge(e Edge, m scene.Measurer) (graphlayout.Edge, label.Box, error) {
	edge := graphlayout.Edge{ID: e.ID, Start: e.Start, End: e.End, StartMarker: e.StartMarker, EndMarker: e.EndMarker}
	if e.Label == "" {
		return edge, label.Box{}, nil
	}
	lines, err := label.ReadEdge(e.Label, "markdown", e.At)
	if err != nil {
		return edge, label.Box{}, err
	}
	text := label.Set(m, lines, edgeWrap, 0)
	edge.Label = sizeOf(&text)
	return edge, text, nil
}

// endText sets the text at one end of an edge, nil when there is nothing to show.
func endText(text string, at syntax.Pos, m scene.Measurer) (*label.Box, error) {
	if text == "" {
		return nil, nil
	}
	lines, err := label.Read(text, "string", at)
	if err != nil {
		return nil, err
	}
	b := label.SetSized(m, lines, math.Inf(1), 0, endTextSize)
	return &b, nil
}

func sizeOf(b *label.Box) *elk.Size {
	if b == nil {
		return nil
	}
	return &elk.Size{Width: b.W, Height: b.H}
}
