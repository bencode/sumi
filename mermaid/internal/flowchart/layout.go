package flowchart

import (
	"math"

	"github.com/bencode/sumi/mermaid/internal/elk"
	"github.com/bencode/sumi/mermaid/internal/graphlayout"
	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/shape"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// The widths labels are set in and the paddings of getData, at Mermaid's defaults: a
// node's label wraps at flowchart.wrappingWidth's node fallback and is no narrower
// than minNodeWidth; an edge's wraps at createText's default.
const (
	nodeWrap     = 120
	nodeLeast    = 120
	edgeWrap     = 200
	groupWrap    = 120 // a Markdown title's, which measureGroupLabel still wraps as labelHelper does
	nodePadding  = 15  // flowchart.padding
	groupPadding = 8
)

// kinds are the shapes the grammar writes, as shape sets them (getTypeFromVertex).
var kinds = map[Shape]shape.Kind{
	Rect: shape.Rect, Round: shape.Rounded, Stadium: shape.Stadium, Subroutine: shape.Subroutine,
	Cylinder: shape.Cylinder, Circle: shape.Circle, DoubleCircle: shape.DoubleCircle, Diamond: shape.Diamond,
	Hexagon: shape.Hexagon, Odd: shape.Odd, Trapezoid: shape.Trapezoid, InvTrapezoid: shape.InvTrapezoid,
	LeanRight: shape.LeanRight, LeanLeft: shape.LeanLeft,
}

// Layout lays out a flowchart's body, its front matter already taken off, and
// draws it; title is the front matter's title, if any.
func Layout(body, title string, m scene.Measurer) (*scene.Scene, error) {
	d, err := Read(body)
	if err != nil {
		return nil, err
	}
	if err := ends(d); err != nil {
		return nil, err
	}
	return draw(place(d, m), title, m), nil
}

// ends is the error ELK's JSON import gives Mermaid for an edge that ends at no
// node: an end named after an edge's id, which addVertex makes no vertex of.
func ends(d *Diagram) error {
	nodes := make(map[string]bool, len(d.Nodes))
	for _, n := range d.Nodes {
		nodes[n.ID] = true
	}
	for _, e := range d.Edges {
		for _, end := range []string{e.Start, e.End} {
			if !nodes[end] {
				return syntax.Errorf(e.At, "the link ends at %q, an edge's id, not a node", end)
			}
		}
	}
	return nil
}

// A placed is a diagram with its labels set and laid out: what draw draws.
type placed struct {
	d      *Diagram
	labels []label.Box  // by d.Nodes: a node's label, a subgraph's title
	shapes []shape.Node // by d.Nodes; zero for a subgraph
	edges  []label.Box  // by d.Edges
	r      graphlayout.Result
}

// place sets every label and shape of d and lays d out, as defaultMeasureLayout and
// runElkLayoutCore do.
func place(d *Diagram, m scene.Measurer) placed {
	p := placed{d: d, labels: make([]label.Box, len(d.Nodes)), shapes: make([]shape.Node, len(d.Nodes)), edges: make([]label.Box, len(d.Edges))}
	data := graphlayout.Data{Direction: elkDirection(d.Direction)}
	for i, n := range d.Nodes {
		node := graphlayout.Node{ID: n.ID, Parent: n.Parent, Group: n.Group, Direction: elk.Undefined}
		if n.Dir != "" {
			node.Direction = elkDirection(n.Dir)
		}
		if n.Group {
			wrap := math.Inf(1)
			if n.Label.Kind == "markdown" {
				wrap = groupWrap
			}
			p.labels[i] = label.Set(m, n.Label.Lines, wrap, 0)
			if n.Label.Text != "" {
				node.Title = &elk.Size{Width: p.labels[i].W, Height: p.labels[i].H}
			}
			node.Padding = groupPadding
		} else {
			k := kinds[n.Shape]
			least := 0.0
			if n.Label.Text != "" && k.WidensLabel() {
				least = nodeLeast
			}
			p.labels[i] = label.Set(m, n.Label.Lines, nodeWrap, least)
			s := shape.Set(k, p.labels[i].W, p.labels[i].H, nodePadding)
			p.shapes[i] = s
			node.Width, node.Height = s.W, s.H
			node.Intersect = func(b graphlayout.Box, to elk.Vec) elk.Vec {
				q := s.Intersect(scene.Point{X: b.X, Y: b.Y}, b.Width, b.Height, scene.Point{X: to.X, Y: to.Y})
				return elk.Vec{X: q.X, Y: q.Y}
			}
		}
		data.Nodes = append(data.Nodes, node)
	}
	for i, e := range d.Edges {
		edge := graphlayout.Edge{ID: e.ID, Start: e.Start, End: e.End, StartMarker: e.StartMarker, EndMarker: e.EndMarker}
		if e.Label.Text != "" {
			p.edges[i] = label.Set(m, e.Label.Lines, edgeWrap, 0)
			edge.Label = &elk.Size{Width: p.edges[i].W, Height: p.edges[i].H}
		}
		data.Edges = append(data.Edges, edge)
	}
	p.r = graphlayout.Layout(data)
	return p
}

// elkDirection is dir2ElkDirection: anything but LR, RL and BT reads as down.
func elkDirection(dir string) elk.Direction {
	switch dir {
	case "LR":
		return elk.Right
	case "RL":
		return elk.Left
	case "BT":
		return elk.Up
	}
	return elk.Down
}
