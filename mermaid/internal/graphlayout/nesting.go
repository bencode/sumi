package graphlayout

import "slices"

// nestedOrder gives the order to hand ELK the edges in. ELK lines up edges that
// share an end in the order of the input, even at the cost of a crossing, so an edge
// from a subgraph's border and a later one from a node inside it, to the same
// target, cross on the way there. Each such edge moves in front of the first edge
// before it that leaves the subgraph around its source for the same target, or
// enters the subgraph around its target from the same source (orderNestedEdgeEnds,
// nestedEdgeOrder.ts).
func nestedOrder(edges []ends, t tree) []int {
	order := make([]int, len(edges))
	for i := range order {
		order[i] = i
	}
	for index, e := range edges {
		from := slices.Index(order, index)
		to := slices.IndexFunc(order[:from], func(other int) bool {
			earlier := edges[other]
			return earlier.target == e.target && t.inside(e.source, earlier.source) ||
				earlier.source == e.source && t.inside(e.target, earlier.target)
		})
		if to != -1 {
			order = slices.Insert(slices.Delete(order, from, from+1), to, index)
		}
	}
	return order
}
