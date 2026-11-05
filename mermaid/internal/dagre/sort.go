package dagre

import (
	"cmp"
	"slices"
)

// A center is a barycenter, the mean place of a node's neighbors, and the
// weight it carries. ok is whether there is one: JS leaves the property
// out when not.
type center struct {
	value  float64
	weight int
	ok     bool
}

// An entry is nodes that move together: vs, in order, and i, the lowest
// place one of them had.
type entry struct {
	vs []string
	i  int
	center
}

// sortSubgraph sorts v's children: by barycenter, a subgraph's children
// first, with subgraphs as cg has them and v's own border nodes at the
// ends (sort-subgraph.js line 8).
func sortSubgraph(lg *layerGraph, v string, cg *constraints, biasRight bool) entry {
	l := lg.node(v)
	movable := slices.DeleteFunc(lg.childrenOf(v), func(w string) bool {
		return l.left != "" && (w == l.left || w == l.right)
	})
	entries := barycenters(lg, movable)
	subgraphs := map[string]entry{}
	for _, e := range entries {
		w := e.vs[0]
		if len(lg.childrenOf(w)) == 0 {
			continue
		}
		sub := sortSubgraph(lg, w, cg, biasRight)
		subgraphs[w] = sub
		if sub.ok {
			mergeBarycenters(&e.center, sub.center)
		}
	}
	entries = resolveConflicts(entries, cg)
	expandSubgraphs(entries, subgraphs)
	result := sortEntries(entries, biasRight)
	if l.left == "" {
		return result
	}
	result.vs = slices.Concat([]string{l.left}, result.vs, []string{l.right})
	if preds := lg.predecessors(l.left); len(preds) > 0 {
		left, right := lg.node(preds[0]).n, lg.node(lg.predecessors(l.right)[0]).n
		sum := result.value*float64(result.weight) + float64(left.order) + float64(right.order)
		result.value = sum / float64(result.weight+2)
		result.weight += 2
		result.ok = true
	}
	return result
}

// barycenters is an entry for each node of movable, with the mean order
// of its neighbors, by edge weight, when it has any (barycenter.js line
// 5).
func barycenters(lg *layerGraph, movable []string) []*entry {
	entries := make([]*entry, len(movable))
	for i, v := range movable {
		e := &entry{vs: []string{v}, i: i}
		if in := lg.inEdges(v); len(in) > 0 {
			sum, weight := 0, 0
			for _, k := range in {
				w := lg.edgeOf(k).weight
				sum += w * lg.node(k.v).n.order
				weight += w
			}
			e.center = center{float64(sum) / float64(weight), weight, true}
		}
		entries[i] = e
	}
	return entries
}

// expandSubgraphs puts each subgraph's sorted nodes in place of it
// (sort-subgraph.js line 55).
func expandSubgraphs(entries []*entry, subgraphs map[string]entry) {
	for _, e := range entries {
		var vs []string
		for _, v := range e.vs {
			if sub, ok := subgraphs[v]; ok {
				vs = append(vs, sub.vs...)
			} else {
				vs = append(vs, v)
			}
		}
		e.vs = vs
	}
}

// mergeBarycenters adds other to target (sort-subgraph.js line 68).
func mergeBarycenters(target *center, other center) {
	if !target.ok {
		*target = other
		return
	}
	sum := target.value*float64(target.weight) + other.value*float64(other.weight)
	target.value = sum / float64(target.weight+other.weight)
	target.weight += other.weight
}

// sortEntries orders entries by barycenter, those without one keeping
// their place, and returns the nodes in that order with their barycenter
// (sort.js line 6). Barycenters are never NaN here, and i differs from
// entry to entry, so the order is total and any sort gives V8's result.
func sortEntries(entries []*entry, biasRight bool) entry {
	var sortable, unsortable []*entry
	for _, e := range entries {
		if e.ok {
			sortable = append(sortable, e)
		} else {
			unsortable = append(unsortable, e)
		}
	}
	slices.SortFunc(unsortable, func(a, b *entry) int { return b.i - a.i })
	slices.SortFunc(sortable, func(a, b *entry) int {
		if c := cmp.Compare(a.value, b.value); c != 0 {
			return c
		}
		if biasRight {
			return b.i - a.i
		}
		return a.i - b.i
	})
	var result entry
	index := 0
	consume := func() { // consumeUnsortable, sort.js line 39
		for len(unsortable) > 0 && unsortable[len(unsortable)-1].i <= index {
			result.vs = append(result.vs, unsortable[len(unsortable)-1].vs...)
			unsortable = unsortable[:len(unsortable)-1]
			index++
		}
	}
	consume()
	sum, weight := 0.0, 0
	for _, e := range sortable {
		index += len(e.vs)
		result.vs = append(result.vs, e.vs...)
		sum += e.value * float64(e.weight)
		weight += e.weight
		consume()
	}
	if weight != 0 {
		result.center = center{sum / float64(weight), weight, true}
	}
	return result
}
