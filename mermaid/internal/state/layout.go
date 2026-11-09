package state

import (
	"math"

	"github.com/bencode/sumi/mermaid/internal/elk"
	"github.com/bencode/sumi/mermaid/internal/graphlayout"
	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// What dataFetcher and the renderers give what Size leaves out, at Mermaid's defaults.
const (
	groupPadding  = 8   // a composite's, which only its title's floor reads
	titleWrap     = 120 // flowchart.wrappingWidth, which measureGroupLabel reads for every diagram
	edgeWrap      = 200 // createText's default
	transitionEnd = "arrow_barb_neo"
)

// Placed is a diagram sized and laid out: what drawing reads.
type Placed struct {
	D      *Diagram
	Sized  []Sized     // by D.Nodes
	Titles []label.Box // by D.Nodes: a Composite's title
	Edges  []label.Box // by D.Edges
	R      graphlayout.Result
}

// Place sizes d and lays it out (defaultMeasureLayout, runElkLayoutCore).
func Place(d *Diagram, m scene.Measurer) (Placed, error) {
	if err := ends(d); err != nil {
		return Placed{}, err
	}
	sized, err := Size(d, m)
	if err != nil {
		return Placed{}, err
	}
	p := Placed{D: d, Sized: sized, Titles: make([]label.Box, len(d.Nodes)), Edges: make([]label.Box, len(d.Edges))}
	data := graphlayout.Data{Direction: graphlayout.ElkDirection(d.Direction), Nodes: make([]graphlayout.Node, len(d.Nodes)), Edges: make([]graphlayout.Edge, len(d.Edges))}
	for i, n := range d.Nodes {
		if data.Nodes[i], p.Titles[i], err = layoutNode(n, sized[i], m); err != nil {
			return Placed{}, err
		}
	}
	for i, e := range d.Edges {
		if data.Edges[i], p.Edges[i], err = layoutEdge(e, m); err != nil {
			return Placed{}, err
		}
	}
	p.R = graphlayout.Layout(data)
	return p, nil
}

// ends is the error ELK's JSON import gives Mermaid for an edge that ends at no node:
// an end named root, which is the diagram itself and no state.
func ends(d *Diagram) error {
	nodes := make(map[string]bool, len(d.Nodes))
	for _, n := range d.Nodes {
		nodes[n.ID] = true
	}
	for _, e := range d.Edges {
		for _, end := range []string{e.Start, e.End} {
			if !nodes[end] {
				return syntax.Errorf(e.At, "the transition ends at %q, which is no state", end)
			}
		}
	}
	return nil
}

// layoutNode is what layout gets of n, and a Composite's title.
func layoutNode(n Node, s Sized, m scene.Measurer) (graphlayout.Node, label.Box, error) {
	node := graphlayout.Node{ID: n.ID, Parent: n.Parent, Group: n.Group, Direction: elk.Undefined}
	if n.Dir != "" {
		node.Direction = graphlayout.ElkDirection(n.Dir)
	}
	if !n.Group {
		node.Width, node.Height, node.Intersect = s.Shape.W, s.Shape.H, graphlayout.ShapeIntersect(s.Shape)
		return node, label.Box{}, nil
	}
	if n.Label == "" { // a region's is cleared
		return node, label.Box{}, nil
	}
	if n.Shape == NoteGroup {
		return noteGroup(node, n, m)
	}
	node.Padding = groupPadding
	lines, err := label.Read(n.Label, "markdown", n.At)
	if err != nil {
		return node, label.Box{}, err
	}
	title := label.Set(m, lines, titleWrap, 0)
	node.Title = &elk.Size{Width: title.W, Height: title.H}
	return node, title, nil
}

// noteGroup gives a note group its label, the note's text. Nothing shows it there, and
// clusterPaintsTitle asks ELK for no size for it, but ELK still leaves its height above
// what the group holds and keeps room for what sticks out at the sides. It has no
// labelType, so unwrapGroupLabels measures it on one line.
func noteGroup(node graphlayout.Node, n Node, m scene.Measurer) (graphlayout.Node, label.Box, error) {
	lines, err := label.Read(n.Label, "string", n.At)
	if err != nil {
		return node, label.Box{}, err
	}
	b := label.Set(m, lines, math.Inf(1), 0)
	node.Title, node.Untitled = &elk.Size{Width: b.W, Height: b.H}, true
	return node, label.Box{}, nil
}

// layoutEdge is what layout gets of e, and its label. A note's line has no arrow.
func layoutEdge(e Edge, m scene.Measurer) (graphlayout.Edge, label.Box, error) {
	edge := graphlayout.Edge{ID: e.ID, Start: e.Start, End: e.End}
	if !e.Note {
		edge.EndMarker = transitionEnd
	}
	if e.Label == "" {
		return edge, label.Box{}, nil
	}
	lines, err := label.ReadEdge(e.Label, "markdown", e.At)
	if err != nil {
		return edge, label.Box{}, err
	}
	box := label.Set(m, lines, edgeWrap, 0)
	edge.Label = &elk.Size{Width: box.W, Height: box.H}
	return edge, box, nil
}
