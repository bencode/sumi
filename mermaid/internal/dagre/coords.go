package dagre

// adjustCoordinates has the steps that follow work from top to bottom,
// whatever rankDir (coordinate-system.js line 5).
func adjustCoordinates(g *layoutGraph, rankDir string) {
	if rankDir == "LR" || rankDir == "RL" {
		swapWidthHeight(g)
	}
}

// undoCoordinates turns the graph to rankDir (coordinate-system.js line
// 12).
func undoCoordinates(g *layoutGraph, rankDir string) {
	if rankDir == "BT" || rankDir == "RL" {
		reverseY(g)
	}
	if rankDir == "LR" || rankDir == "RL" {
		swapXY(g)
		swapWidthHeight(g)
	}
}

// swapWidthHeight is coordinate-system.js line 24.
func swapWidthHeight(g *layoutGraph) {
	for _, v := range g.nodeNames() {
		n := g.node(v)
		n.width, n.height = n.height, n.width
	}
	for _, e := range g.edgeList() {
		l := g.edgeOf(e)
		l.width, l.height = l.height, l.width
	}
}

// reverseY is coordinate-system.js line 39.
func reverseY(g *layoutGraph) {
	for _, v := range g.nodeNames() {
		n := g.node(v)
		n.y = -n.y
	}
	for _, e := range g.edgeList() {
		l := g.edgeOf(e)
		for i := range l.points {
			l.points[i].Y = -l.points[i].Y
		}
		if l.hasXY {
			l.y = -l.y
		}
	}
}

// swapXY is coordinate-system.js line 57.
func swapXY(g *layoutGraph) {
	for _, v := range g.nodeNames() {
		n := g.node(v)
		n.x, n.y = n.y, n.x
	}
	for _, e := range g.edgeList() {
		l := g.edgeOf(e)
		for i, p := range l.points {
			l.points[i] = Point{p.Y, p.X}
		}
		if l.hasXY {
			l.x, l.y = l.y, l.x
		}
	}
}
