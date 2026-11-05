// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "slices"

// edgeThickness is ELK's default, which Mermaid leaves alone (core Core.melk
// line 1031). A dummy node is as tall as the edge it carries.
const edgeThickness = 1

// splitLongEdges cuts every edge crossing a layer into one edge per pair of
// layers, with a dummy node in each layer between. An edge's new second half
// is cut in turn when its layer comes up (process,
// intermediate/LongEdgeSplitter.java line 63).
func splitLongEdges(g *Graph) {
	layerOf := map[*Node]int{}
	for i, l := range g.Layers {
		for _, n := range l.Nodes {
			layerOf[n] = i
		}
	}
	for i := 0; i+1 < len(g.Layers); i++ {
		for _, n := range g.Layers[i].Nodes {
			for _, e := range outgoing(n) {
				if target := layerOf[e.Target.Owner]; target != i && target != i+1 {
					splitEdge(e, newDummy(g.Layers[i+1]))
				}
			}
		}
	}
}

// newDummy adds a dummy node to l (createDummyNode,
// LongEdgeSplitter.java line 113).
func newDummy(l *Layer) *Node {
	n := &Node{Type: typeLongEdge}
	l.Nodes = append(l.Nodes, n)
	return n
}

// splitEdge ends e at dummy and adds an edge from dummy to e's old end. dummy's
// ports sit at half the thickness rounded down, which is 0
// (LongEdgeSplitter.java line 136).
func splitEdge(e *Edge, dummy *Node) {
	end := e.Target
	dummy.Size.Y = edgeThickness
	in := &Port{Owner: dummy, Side: sideWest}
	out := &Port{Owner: dummy, Side: sideEast}
	dummy.Ports = []*Port{in, out}
	setTarget(e, in)
	rest := &Edge{ID: e.ID, ModelOrder: e.ModelOrder, Reversed: e.Reversed}
	setSource(rest, out)
	setTarget(rest, end)
}

// joinLongEdges undoes splitLongEdges (process,
// intermediate/LongEdgeJoiner.java line 66).
func joinLongEdges(g *Graph) {
	isDummy := func(n *Node) bool { return n.Type == typeLongEdge }
	for _, l := range g.Layers {
		for _, n := range l.Nodes {
			if isDummy(n) {
				joinAt(n)
			}
		}
		l.Nodes = slices.DeleteFunc(l.Nodes, isDummy)
	}
}

// joinAt lets the edge into n run on to where the edge out of n ends. n leaves
// a bend behind even on a straight edge, as Mermaid sets unnecessaryBendpoints.
// A port holds one edge, so ELK's care for the order of a port's edges falls
// away, and neither half has labels: the label dummy nodes hold them
// (LongEdgeJoiner.java line 104).
func joinAt(n *Node) {
	kept, dropped := portOn(n, sideWest).In[0], portOn(n, sideEast).Out[0]
	end := dropped.Target
	setSource(dropped, nil)
	setTarget(dropped, nil)
	setTarget(kept, end)
	kept.Bends = append(append(kept.Bends, absolute(n.Ports[0])), dropped.Bends...)
}

func portOn(n *Node, side PortSide) *Port {
	return n.Ports[slices.IndexFunc(n.Ports, func(p *Port) bool { return p.Side == side })]
}
