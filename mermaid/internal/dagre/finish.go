package dagre

import (
	"math"
	"slices"
)

// removeBorderNodes gives each subgraph the box its border nodes make, and
// removes them (layout.js line 309).
func removeBorderNodes(g *layoutGraph) {
	for _, v := range g.nodeNames() {
		if !hasChildren(g, v) {
			continue
		}
		n := g.node(v)
		t, b := g.node(n.borderTop), g.node(n.borderBottom)
		l := g.node(n.borderLeft[len(n.borderLeft)-1])
		r := g.node(n.borderRight[len(n.borderRight)-1])
		n.width, n.height = math.Abs(r.x-l.x), math.Abs(b.y-t.y)
		n.x, n.y = l.x+n.width/2, t.y+n.height/2
	}
	for _, v := range g.nodeNames() {
		if g.node(v).dummy == "border" {
			g.removeNode(v)
		}
	}
}

// translateGraph moves the graph so that what it bounds starts at the
// margins, and returns the graph's size. It bounds nodes and labels, but
// not edges' bends. Its right and bottom never come before 0 (layout.js
// line 205).
func translateGraph(g *layoutGraph, marginX, marginY float64) (width, height float64) {
	minX, maxX, minY, maxY := math.Inf(1), 0.0, math.Inf(1), 0.0
	extremes := func(x, y, w, h float64) {
		minX, maxX = min(minX, x-w/2), max(maxX, x+w/2)
		minY, maxY = min(minY, y-h/2), max(maxY, y+h/2)
	}
	for _, v := range g.nodeNames() {
		n := g.node(v)
		extremes(n.x, n.y, n.width, n.height)
	}
	for _, e := range g.edgeList() {
		if l := g.edgeOf(e); l.hasXY {
			extremes(l.x, l.y, l.width, l.height)
		}
	}
	minX -= marginX
	minY -= marginY
	for _, v := range g.nodeNames() {
		n := g.node(v)
		n.x -= minX
		n.y -= minY
	}
	for _, e := range g.edgeList() {
		l := g.edgeOf(e)
		for i := range l.points {
			l.points[i].X -= minX
			l.points[i].Y -= minY
		}
		if l.hasXY {
			l.x -= minX
			l.y -= minY
		}
	}
	return maxX - minX + marginX, maxY - minY + marginY
}

// assignNodeIntersects starts and ends each edge at the edge of its nodes
// (layout.js line 262).
func assignNodeIntersects(g *layoutGraph) {
	for _, e := range g.edgeList() {
		l, v, w := g.edgeOf(e), g.node(e.v), g.node(e.w)
		first, last := Point{w.x, w.y}, Point{v.x, v.y}
		if l.points != nil {
			first, last = l.points[0], l.points[len(l.points)-1]
		}
		l.points = slices.Concat([]Point{intersectRect(v, first)}, l.points, []Point{intersectRect(w, last)})
	}
}

// intersectRect is where the line from n's center to p leaves n (util.js
// line 94). With rankSep above 0, bends and nodes in different ranks are
// apart, so p is never the center.
func intersectRect(n *node, p Point) Point {
	dx, dy := p.X-n.x, p.Y-n.y
	w, h := n.width/2, n.height/2
	if dx == 0 && dy == 0 {
		panic("dagre: no intersection from a node's own center")
	}
	if math.Abs(dy)*w > math.Abs(dx)*h {
		if dy < 0 {
			h = -h
		}
		return Point{n.x + h*dx/dy, n.y + h}
	}
	if dx < 0 {
		w = -w
	}
	return Point{n.x + w, n.y + w*dy/dx}
}

// reversePoints runs the paths of the edges acyclic turned around the way
// they go (layout.js line 300).
func reversePoints(g *layoutGraph) {
	for _, e := range g.edgeList() {
		if l := g.edgeOf(e); l.reversed {
			slices.Reverse(l.points)
		}
	}
}
