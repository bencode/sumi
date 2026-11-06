// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "slices"

// invertPorts puts a long edge dummy in the layer of a subgraph, next to each port
// whose edge comes in on the east side or leaves on the west, so that the edge
// goes round to the side it can reach. Those dummies are the only source of edges
// within a layer. Only ports on fixed sides can be the wrong way round (process,
// intermediate/InvertedPortProcessor.java line 84).
func invertPorts(g *Graph) {
	for _, l := range g.Layers {
		var added []*Node
		for _, n := range l.Nodes {
			if !n.FixedPorts {
				continue
			}
			for _, p := range n.Ports {
				if p.Side == sideEast {
					added = append(added, splitInverted(p.In)...)
				}
			}
			for _, p := range n.Ports {
				if p.Side == sideWest {
					added = append(added, splitInverted(p.Out)...)
				}
			}
		}
		l.Nodes = append(l.Nodes, added...)
	}
}

// splitInverted cuts the edges given (createEastPortSideDummies and
// createWestPortSideDummies, InvertedPortProcessor.java lines 171 and 241). Unlike
// LongEdgeSplitter, ELK leaves these dummies without height.
func splitInverted(all []*Edge) []*Node {
	var dummies []*Node
	for _, e := range slices.Clone(all) {
		dummy := &Node{Type: typeLongEdge}
		splitEdge(e, dummy)
		dummy.Size.Y = 0
		dummies = append(dummies, dummy)
	}
	return dummies
}
