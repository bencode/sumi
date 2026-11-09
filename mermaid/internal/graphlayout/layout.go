package graphlayout

import (
	"slices"

	"github.com/bencode/sumi/mermaid/internal/elk"
)

// Layout places d (runElkLayoutCore, applyElkLayoutResult, render.ts).
func Layout(d Data) Result {
	in, reversed := elkInput(d)
	out := elk.Layout(in)
	t := treeOf(d.Nodes)
	edges := zipEdges(in.Edges, out.Edges)
	frames := placeNodes(out.Nodes, d.Nodes)
	frames = evenFrames(out.Nodes, frames, edges, t)
	frames = alignSmallNodes(frames, edges, t)
	routes := separateLabels(placeTexts(straightenEdges(routeEdges(d.Edges, edges, reversed, frames, t)), frames))
	return Result{placedNodes(d.Nodes, frames, t), placedEdges(routes)}
}

// elkEdge is an edge as ELK had it and what it made of it. Its points are relative
// to the edge's container.
type elkEdge struct {
	id, source, target string
	points             []elk.Vec
	labels             []elk.PlacedLabel
	ends               [2]*elk.PlacedLabel // its tail and head labels
}

func zipEdges(in []elk.InputEdge, out []elk.PlacedEdge) []elkEdge {
	edges := make([]elkEdge, len(in))
	for i, e := range in {
		edges[i] = elkEdge{e.ID, e.Source, e.Target, out[i].Points, out[i].Labels, [2]*elk.PlacedLabel{out[i].Tail, out[i].Head}}
	}
	return edges
}

// placedNodes gives the nodes to paint, subgraphs first, the outer ones before the
// inner, as a subgraph's frame goes under what it holds (orderNodesForElkPaint,
// render.ts).
func placedNodes(nodes []Node, frames map[string]frame, t tree) []PlacedNode {
	ordered := slices.Clone(nodes)
	slices.SortStableFunc(ordered, func(a, b Node) int {
		switch {
		case a.Group != b.Group && a.Group:
			return -1
		case a.Group != b.Group:
			return 1
		case a.Group:
			return depth(t, a.ID) - depth(t, b.ID)
		}
		return 0
	})
	placed := make([]PlacedNode, len(ordered))
	for i, n := range ordered {
		f := frames[n.ID]
		center := f.center()
		placed[i] = PlacedNode{n.ID, Box{center.X, center.Y, f.drawnWidth, f.height}}
	}
	return placed
}

// depth counts what holds id (getGroupDepth, render.ts).
func depth(t tree, id string) int {
	n := 0
	for p := t[id]; p != ""; p = t[p] {
		n++
	}
	return n
}

func placedEdges(routes []route) []PlacedEdge {
	placed := make([]PlacedEdge, len(routes))
	for i, r := range routes {
		placed[i] = PlacedEdge{ID: r.id, Points: r.points}
		if r.label != nil {
			placed[i].Label = &r.label.center
		}
		if r.texts[0] != nil {
			placed[i].Start = &r.texts[0].center
		}
		if r.texts[1] != nil {
			placed[i].End = &r.texts[1].center
		}
	}
	return placed
}
