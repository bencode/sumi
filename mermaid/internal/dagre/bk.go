package dagre

import "slices"

// conflicts are the edges that may not be drawn straight, by their ends in
// the order JS compares them.
type conflicts map[[2]string]bool

func (c conflicts) add(v, w string)      { c[conflictKey(v, w)] = true }
func (c conflicts) has(v, w string) bool { return c[conflictKey(v, w)] }

// conflictKey is v and w with the smaller first (bk.js lines 142 and 167).
func conflictKey(v, w string) [2]string {
	if jsLess(w, v) {
		v, w = w, v
	}
	return [2]string{v, w}
}

// findType1Conflicts adds the edges that cross an inner segment, an edge
// between two dummies, to c. It scans each layer up to each node on such a
// segment, and takes the edges into the nodes it passed from outside the
// two segments around them (bk.js line 40).
func findType1Conflicts(g *layoutGraph, layering [][]string, c conflicts) {
	for r := 1; r < len(layering); r++ {
		prev, layer := layering[r-1], layering[r]
		k0, scanPos := 0, 0
		for i, v := range layer {
			w := findOtherInnerSegmentNode(g, v)
			if w == "" && i != len(layer)-1 {
				continue
			}
			k1 := len(prev)
			if w != "" {
				k1 = g.node(w).order
			}
			for _, scan := range layer[scanPos : i+1] {
				for _, u := range g.predecessors(scan) {
					l := g.node(u)
					if (l.order < k0 || k1 < l.order) && !(l.dummy != "" && g.node(scan).dummy != "") {
						c.add(u, scan)
					}
				}
			}
			scanPos, k0 = i+1, k1
		}
	}
}

// findType2Conflicts adds the inner segments that go across a subgraph's
// border to c (bk.js line 81). nextNorth is undefined in JS until the
// first border, and order < undefined is false, as order < -1 is.
func findType2Conflicts(g *layoutGraph, layering [][]string, c conflicts) {
	scan := func(south []string, from, to, prevBorder, nextBorder int) {
		for _, v := range south[from:to] {
			if g.node(v).dummy == "" {
				continue
			}
			for _, u := range g.predecessors(v) {
				l := g.node(u)
				if l.dummy != "" && (l.order < prevBorder || l.order > nextBorder) {
					c.add(u, v)
				}
			}
		}
	}
	for r := 1; r < len(layering); r++ {
		north, south := layering[r-1], layering[r]
		prevNorth, nextNorth, southPos := -1, -1, 0
		for lookahead, v := range south {
			if preds := g.predecessors(v); g.node(v).dummy == "border" && len(preds) > 0 {
				nextNorth = g.node(preds[0]).order
				scan(south, southPos, lookahead, prevNorth, nextNorth)
				southPos, prevNorth = lookahead, nextNorth
			}
			scan(south, southPos, len(south), nextNorth, len(north))
		}
	}
}

// findOtherInnerSegmentNode is the dummy before v when v is a dummy too,
// "" when there is none (bk.js line 126).
func findOtherInnerSegmentNode(g *layoutGraph, v string) string {
	if g.node(v).dummy == "" {
		return ""
	}
	for _, u := range g.predecessors(v) {
		if g.node(u).dummy != "" {
			return u
		}
	}
	return ""
}

// verticalAlignment joins each node to a median neighbor, where no conflict
// forbids it and the edge would not cross one joined before, into blocks:
// root is each node's block, and align the next node in it, going round
// (bk.js line 183). Places are those in layering, which may be turned over.
func verticalAlignment(layering [][]string, c conflicts, neighbors func(string) []string) (root, align map[string]string) {
	root, align = map[string]string{}, map[string]string{}
	pos := map[string]int{}
	for _, layer := range layering {
		for i, v := range layer {
			root[v], align[v], pos[v] = v, v, i
		}
	}
	for _, layer := range layering {
		prevIdx := -1
		for _, v := range layer {
			ws := neighbors(v)
			if len(ws) == 0 {
				continue
			}
			slices.SortStableFunc(ws, func(a, b string) int { return pos[a] - pos[b] })
			for i := (len(ws) - 1) / 2; i <= len(ws)/2; i++ {
				w := ws[i]
				if align[v] == v && prevIdx < pos[w] && !c.has(v, w) {
					align[w] = v
					root[v] = root[w]
					align[v] = root[v]
					prevIdx = pos[w]
				}
			}
		}
	}
	return root, align
}
