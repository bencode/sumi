package dagre

import "slices"

// crossCount is how many crossings layering has, each counting the
// product of its two edges' weights (cross-count.js line 21).
func crossCount(g *layoutGraph, layering [][]string) int {
	cc := 0
	for i := 1; i < len(layering); i++ {
		cc += twoLayerCrossCount(g, layering[i-1], layering[i])
	}
	return cc
}

// twoLayerCrossCount counts the crossings between two ranks. With the
// edges sorted by their place in north, then in south, two edges cross
// when the later one ends left of the earlier. An accumulator tree over
// south's places adds that up as each edge comes (cross-count.js line 29,
// after Barth et al., "Bilayer Cross Counting").
func twoLayerCrossCount(g *layoutGraph, north, south []string) int {
	type end struct{ pos, weight int }
	places := make(map[string]int, len(south))
	for i, v := range south {
		places[v] = i
	}
	var ends []end
	for _, v := range north {
		from := len(ends)
		for _, e := range g.outEdges(v) {
			ends = append(ends, end{places[e.w], g.edgeOf(e).weight})
		}
		slices.SortStableFunc(ends[from:], func(a, b end) int { return a.pos - b.pos })
	}
	first := 1
	for first < len(south) {
		first <<= 1
	}
	tree := make([]int, 2*first-1)
	first--
	cc := 0
	for _, e := range ends {
		i := e.pos + first
		tree[i] += e.weight
		sum := 0
		for i > 0 {
			if i%2 == 1 {
				sum += tree[i+1]
			}
			i = (i - 1) >> 1
			tree[i] += e.weight
		}
		cc += e.weight * sum
	}
	return cc
}
