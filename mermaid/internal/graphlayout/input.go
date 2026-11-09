package graphlayout

import (
	"github.com/bencode/sumi/mermaid/internal/elk"
	"github.com/bencode/sumi/mermaid/internal/shape"
	"github.com/bencode/sumi/scene"
)

// ElkDirection is dir2ElkDirection: anything but LR, RL and BT reads as down.
func ElkDirection(dir string) elk.Direction {
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

// ShapeIntersect is where a line meets the outline of s.
func ShapeIntersect(s shape.Node) Intersect {
	return func(b Box, to elk.Vec) elk.Vec {
		q := s.Intersect(scene.Point{X: b.X, Y: b.Y}, b.Width, b.Height, scene.Point{X: to.X, Y: to.Y})
		return elk.Vec{X: q.X, Y: q.Y}
	}
}

// elkInput gives what Mermaid hands ELK for d, and the IDs of the edges it
// turned round (buildElkGraphFromLayoutData, render.ts). Mermaid's options on the
// root and on subgraphs are in the elk package.
func elkInput(d Data) (elk.Input, map[string]bool) {
	t := treeOf(d.Nodes)
	children := map[string][]Node{}
	for _, n := range d.Nodes {
		children[n.Parent] = append(children[n.Parent], n)
	}
	direction := d.Direction
	if direction == elk.Undefined {
		direction = elk.Down
	}
	edges, reversed := elkEdges(d.Edges, t)
	nodes := elkNodes(children[""], children, hierarchies(d, t, children))
	return elk.Input{Direction: direction, Nodes: nodes, Edges: edges}, reversed
}

// elkNodes gives siblings and all below them (addVertices, configureSubgraphNodes,
// render.ts). A subgraph without nodes is left as a plain node of no size.
func elkNodes(siblings []Node, children map[string][]Node, h map[string]elk.Hierarchy) []elk.InputNode {
	all := make([]elk.InputNode, len(siblings))
	for i, n := range siblings {
		all[i] = elk.InputNode{ID: n.ID, Width: n.Width, Height: n.Height, Hierarchy: h[n.ID]}
		if kids := children[n.ID]; len(kids) > 0 {
			all[i].Children = elkNodes(kids, children, h)
			all[i].Direction = n.Direction
			all[i].Title, all[i].MinWidth = title(n)
			all[i].Untitled = n.Untitled
		}
	}
	return all
}

// title gives a subgraph's title and the width that keeps room for it
// (getMeasuredLabelData, groupTitleWidth, render.ts). Mermaid takes 2 off the
// measured height.
func title(n Node) (*elk.Size, float64) {
	if n.Title == nil {
		return nil, n.Padding
	}
	size := &elk.Size{Width: n.Title.Width, Height: n.Title.Height - 2}
	if n.Untitled {
		return size, 0
	}
	return size, n.Title.Width + n.Padding
}

// hierarchies gives how each node's subgraph relates to its parent. One with a
// direction of its own is laid out on its own, unless an edge crosses its border
// (buildSubgraphLayoutOptions, configureCrossHierarchyEdges, render.ts).
func hierarchies(d Data, t tree, children map[string][]Node) map[string]elk.Hierarchy {
	h := map[string]elk.Hierarchy{}
	for _, n := range d.Nodes {
		if n.Direction != elk.Undefined && len(children[n.ID]) > 0 {
			h[n.ID] = elk.Separate
		}
	}
	for _, e := range d.Edges {
		if t[e.Start] != t[e.End] {
			ancestor := t.commonAncestor(e.Start, e.End)
			include(h, t, e.Start, ancestor)
			include(h, t, e.End, ancestor)
		}
	}
	return h
}

// include joins id and what holds it, up to ancestor, in one layout
// (setIncludeChildrenPolicy, render.ts).
func include(h map[string]elk.Hierarchy, t tree, id, ancestor string) {
	for {
		h[id] = elk.Include
		if id == ancestor || t[id] == "" {
			return
		}
		id = t[id]
	}
}

// elkEdges gives edges as ELK gets them, the feedback ones turned round and then
// all in the order nestedOrder wants, and the IDs of those it turned
// (reverseSubgraphFeedbackEdges, orderEdgesByNesting, render.ts).
func elkEdges(edges []Edge, t tree) ([]elk.InputEdge, map[string]bool) {
	directed := make([]ends, len(edges))
	for i, e := range edges {
		directed[i] = ends{e.Start, e.End}
	}
	reversed := map[string]bool{}
	for i, feedback := range feedbackEdges(directed, t) {
		if feedback {
			directed[i] = ends{edges[i].End, edges[i].Start}
			reversed[edges[i].ID] = true
		}
	}
	all := make([]elk.InputEdge, len(edges))
	for i, index := range nestedOrder(directed, t) {
		e := directed[index]
		all[i] = elk.InputEdge{
			ID: edges[index].ID, Source: e.source, Target: e.target, Label: edges[index].Label,
			Tail: endRoom(edges[index].StartText, edges[index].StartMarker),
			Head: endRoom(edges[index].EndText, edges[index].EndMarker),
		}
	}
	return all, reversed
}
