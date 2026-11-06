package graphlayout

// ends is an edge as far as its order and direction go (EdgeEnds,
// subgraphFeedbackEdges.ts).
type ends struct{ source, target string }

// levelEdge is an edge seen in the graph of its lowest common subgraph, between
// two children of it. It is collapsed when a subgraph stands for one of its ends.
type levelEdge struct {
	index     int
	from, to  string
	collapsed bool
}

// feedbackEdges tells which edges close a cycle only because a subgraph is one node
// of its parent: with A -> B inside G and B -> X -> C out of it and back, G and X
// form a cycle though the nodes do not. ELK would turn such an edge round and route
// it around the subgraph, so Mermaid hands it over the other way round. Each edge
// counts in the graph of its lowest common subgraph. A self loop, or an edge between
// a subgraph and its own member, has that subgraph as an end, so it crosses no
// border and is left out (findSubgraphFeedbackEdges, subgraphFeedbackEdges.ts).
func feedbackEdges(edges []ends, t tree) []bool {
	levels := map[string][]levelEdge{}
	for i, e := range edges {
		ancestor := t.commonAncestor(e.source, e.target)
		if ancestor == e.source || ancestor == e.target {
			continue
		}
		from, to := t.childOnPath(e.source, ancestor), t.childOnPath(e.target, ancestor)
		levels[ancestor] = append(levels[ancestor], levelEdge{i, from, to, from != e.source || to != e.target})
	}
	flags := make([]bool, len(edges))
	for _, level := range levels {
		markFeedback(level, flags)
	}
	return flags
}

// markFeedback searches level depth first, from its sources and then from whatever
// is left, in the order of its edges. An edge back to a node on the stack is
// feedback. One between plain nodes is left to ELK, which can then turn that edge
// and keep those into the subgraph pointing downstream.
func markFeedback(level []levelEdge, flags []bool) {
	successors := map[string][]levelEdge{}
	inDegree := map[string]int{}
	var nodes []string
	for _, e := range level {
		for _, n := range []string{e.from, e.to} {
			if _, known := inDegree[n]; !known {
				inDegree[n] = 0
				nodes = append(nodes, n)
			}
		}
		successors[e.from] = append(successors[e.from], e)
		inDegree[e.to]++
	}

	onStack, seen := map[string]bool{}, map[string]bool{}
	var visit func(n string)
	visit = func(n string) {
		seen[n], onStack[n] = true, true
		for _, e := range successors[n] {
			if onStack[e.to] {
				flags[e.index] = e.collapsed
			} else if !seen[e.to] {
				visit(e.to)
			}
		}
		onStack[n] = false
	}
	for _, n := range nodes {
		if inDegree[n] == 0 && !seen[n] {
			visit(n)
		}
	}
	for _, n := range nodes {
		if !seen[n] {
			visit(n)
		}
	}
}
