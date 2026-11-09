// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "slices"

// breakCycles reverses the edges that close a cycle, so that all edges can
// point forward (process and dfs, p1cycles/DepthFirstCycleBreaker.java lines
// 80, 133). No self loop reaches it: SelfLoopPreProcessor takes them out first.
func breakCycles(g *Graph) {
	visited, active := map[*Node]bool{}, map[*Node]bool{}
	var back []*Edge
	var visit func(n *Node)
	visit = func(n *Node) {
		if visited[n] {
			return
		}
		visited[n], active[n] = true, true
		for _, e := range outgoing(n) {
			if target := e.Target.Owner; active[target] {
				back = append(back, e)
			} else {
				visit(target)
			}
		}
		active[n] = false
	}
	for _, n := range g.Layerless {
		if !hasIncoming(n) {
			visit(n)
		}
	}
	for _, n := range g.Layerless {
		visit(n)
	}
	for _, e := range back {
		reverse(e)
	}
}

func hasIncoming(n *Node) bool {
	for _, p := range n.Ports {
		if len(p.In) > 0 {
			return true
		}
	}
	return false
}

// reverse swaps e's ends, and with them its head and tail labels. Their ports
// keep their sides. Collector ports do not come up, as Mermaid sets mergeEdges to
// false (graph/LEdge.java line 58).
func reverse(e *Edge) {
	source, target := e.Source, e.Target
	setSource(e, nil)
	setTarget(e, nil)
	setSource(e, target)
	setTarget(e, source)
	e.Reversed = !e.Reversed
	slices.Reverse(e.Bends)
	for _, l := range e.Labels {
		l.Kind = oppositeEnd[l.Kind]
	}
}

// restoreReversed turns the edges breakCycles reversed back
// (process, intermediate/ReversedEdgeRestorer.java line 46).
func restoreReversed(g *Graph) {
	for _, e := range edges(g) {
		if e.Reversed {
			reverse(e)
		}
	}
}
