package graphlayout

import (
	"maps"
	"math"
	"slices"

	"github.com/bencode/sumi/mermaid/internal/elk"
)

const (
	subgraphPadding = 24 // SUBGRAPH_PADDING
	portsMargin     = 12 // PORTS_SURROUNDING_MARGIN
)

// frame is a node at a stage of placing it (NodeWithVertex). Mermaid keeps a second
// width for painting, as a subgraph's frame may have to fit its title.
type frame struct {
	id                  string
	group               bool
	x, y, width, height float64 // the top left corner; offset.posX, offset.posY
	origin              elk.Vec // where ELK put it, which edges inside it go by
	drawnWidth          float64
	titleFloor          float64 // groupTitleWidth
	intersect           Intersect
}

func (f frame) center() elk.Vec { return elk.Vec{X: f.x + f.width/2, Y: f.y + f.height/2} }

// bounds gives what edges are clipped to (boundsFor, render.ts). It keeps the center
// of the frame while taking the title's width, as Mermaid does.
func (f frame) bounds() Box {
	c, width := f.center(), f.width
	if f.group {
		width = max(width, f.titleFloor)
	}
	return Box{c.X, c.Y, width, f.height}
}

// placeNodes gives each node its absolute place (applyElkNodePositions, render.ts).
func placeNodes(placed []elk.PlacedNode, nodes []Node) map[string]frame {
	byID := map[string]Node{}
	for _, n := range nodes {
		byID[n.ID] = n
	}
	frames := map[string]frame{}
	var walk func(all []elk.PlacedNode, at elk.Vec)
	walk = func(all []elk.PlacedNode, at elk.Vec) {
		for _, p := range all {
			n := byID[p.ID]
			pos := elk.Vec{X: p.X + at.X, Y: p.Y + at.Y}
			f := frame{id: p.ID, group: n.Group, x: pos.X, y: pos.Y, width: p.Width, height: p.Height,
				origin: pos, drawnWidth: p.Width, intersect: n.Intersect}
			if n.Group {
				var titleWidth float64
				if n.Title != nil && !n.Untitled {
					titleWidth = n.Title.Width
				}
				f.drawnWidth = max(p.Width, titleWidth)
				f.titleFloor = titleWidth + n.Padding
			}
			frames[p.ID] = f
			walk(p.Children, pos)
		}
	}
	walk(placed, elk.Vec{})
	return frames
}

// evenFrames pulls each subgraph's frame in to subgraphPadding from what it holds,
// the deepest first. ELK sizes a subgraph around an edge routed along it too, which
// leaves space on one side that nothing visible fills; the edge keeps its lane,
// outside the frame (evenGroupFrames, render.ts).
func evenFrames(placed []elk.PlacedNode, frames map[string]frame, edges []elkEdge, t tree) map[string]frame {
	even := maps.Clone(frames)
	var walk func(all []elk.PlacedNode)
	walk = func(all []elk.PlacedNode) {
		for _, p := range all {
			if !even[p.ID].group {
				continue
			}
			walk(p.Children)
			even[p.ID] = evenFrame(even[p.ID], p.Children, even, edges, t)
		}
	}
	walk(placed)
	return even
}

// evenFrame pulls f in, never out. Its top carries the title and stays.
func evenFrame(f frame, children []elk.PlacedNode, frames map[string]frame, edges []elkEdge, t tree) frame {
	var xs, ys []float64
	for _, c := range children {
		if b := frames[c.ID]; b.width != 0 && b.height != 0 {
			xs = append(xs, b.x, b.x+b.width)
			ys = append(ys, b.y, b.y+b.height)
		}
	}
	if len(xs) == 0 {
		return f
	}
	for _, p := range internalEdgePoints(f.id, edges, frames, t) {
		xs, ys = append(xs, p.X), append(ys, p.Y)
	}
	left := max(f.x, slices.Min(xs)-subgraphPadding)
	right := min(f.x+f.width, slices.Max(xs)+subgraphPadding)
	bottom := min(f.y+f.height, slices.Max(ys)+subgraphPadding)
	x, width := left, right-left
	if width < f.titleFloor {
		x -= (f.titleFloor - width) / 2
		width = f.titleFloor
		x = max(f.x, min(x, f.x+f.width-width))
		width = min(width, f.width)
	}
	f.x, f.width, f.height, f.drawnWidth = x, width, bottom-f.y, width
	return f
}

// internalEdgePoints gives the absolute points of the edges with both ends inside
// id, whose lanes belong to it, and the ends of those attached to its frame, which
// must not be left outside it. The rest of an edge with an end outside belongs to
// the layout around id (internalEdgePoints, render.ts).
func internalEdgePoints(id string, edges []elkEdge, frames map[string]frame, t tree) []elk.Vec {
	var points []elk.Vec
	for _, e := range edges {
		internal := t.inside(e.source, id) && t.inside(e.target, id)
		if !internal && e.source != id && e.target != id {
			continue
		}
		offset := edgeOffset(e.source, e.target, frames, t)
		for i, p := range e.points {
			atStart := i == 0 && e.source == id
			atEnd := i == len(e.points)-1 && e.target == id
			if internal || atStart || atEnd {
				points = append(points, elk.Vec{X: p.X + offset.X, Y: p.Y + offset.Y})
			}
		}
	}
	return points
}

// edgeOffset gives where the points of an edge between two nodes are relative to:
// their lowest common subgraph, or the parent of a self loop's node. Moving a frame
// does not change it (calcOffset, render.ts).
func edgeOffset(start, end string, frames map[string]frame, t tree) elk.Vec {
	ancestor := t.commonAncestor(start, end)
	if start == end {
		ancestor = t[start]
	}
	if ancestor == "" {
		return elk.Vec{}
	}
	return frames[ancestor].origin
}

// alignSmallNodes moves a node too small for ELK to spread ports along a side onto
// the line of the first edge reaching that side. ELK leaves portsMargin free at both
// ends of a side, so on a side shorter than that it parks the port off center, and
// the edge is straight but the node is beside it (alignDegenerateNodeToAnchor,
// render.ts).
func alignSmallNodes(frames map[string]frame, edges []elkEdge, t tree) map[string]frame {
	aligned := maps.Clone(frames)
	done := map[string]bool{}
	align := func(id string, anchor elk.Vec) {
		f := aligned[id]
		alongWidth := math.Abs(anchor.Y-f.y) <= borderTolerance || math.Abs(anchor.Y-(f.y+f.height)) <= borderTolerance
		side := f.height
		if alongWidth {
			side = f.width
		}
		if side >= 2*portsMargin || done[id] {
			return
		}
		done[id] = true
		c := f.center()
		if delta := anchor.X - c.X; alongWidth && math.Abs(delta) >= 0.01 {
			f.x += delta
		} else if delta := anchor.Y - c.Y; !alongWidth && math.Abs(delta) >= 0.01 {
			f.y += delta
		}
		aligned[id] = f
	}
	for _, e := range edges {
		offset := edgeOffset(e.source, e.target, aligned, t)
		first, last := e.points[0], e.points[len(e.points)-1]
		align(e.source, elk.Vec{X: first.X + offset.X, Y: first.Y + offset.Y})
		align(e.target, elk.Vec{X: last.X + offset.X, Y: last.Y + offset.Y})
	}
	return aligned
}
