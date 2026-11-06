// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

// hierarchicalSweepiness is the option's default (Layered.melk line 385).
const hierarchicalSweepiness = 0.1

// pathInfo counts what reaches a node along edges (NodeInfo,
// p3order/LayerSweepTypeDecider.java line 159).
type pathInfo struct{ connected, hierarchical, random int }

// useBottomUp tells whether g orders itself and hands its port order up, or the
// graph around it sweeps through it (useBottomUp, LayerSweepTypeDecider.java line
// 66). Paths through g vote by their ends: an external port for the sweep through,
// a node with nothing before or after it for g itself. An edge within a layer to a
// node already passed carries nothing on, so the order of the layers counts. Nothing
// has fixed a port order yet, and the north and south cases do not come up. ELK adds
// up paths at east ports only, but nothing comes before a west one to add.
func useBottomUp(g *Graph, layers [][]*Node) bool {
	if !included(g) || fewerThanTwoPorts(g.Parent) {
		return true
	}
	infos := map[*Node]*pathInfo{}
	info := func(n *Node) *pathInfo {
		if infos[n] == nil {
			infos[n] = &pathInfo{}
		}
		return infos[n]
	}
	toRandom, toHierarchical := 0, 0
	for _, layer := range layers {
		for _, n := range layer {
			current := info(n)
			switch {
			case n.Type == typeExternalPort:
				current.hierarchical = 1
				toHierarchical += current.connected
			case noEdgesOn(n, sideWest):
				current.random = 1
			case noEdgesOn(n, sideEast):
				toRandom += current.connected
			}
			for _, e := range outgoing(n) {
				toRandom += current.random
				toHierarchical += current.hierarchical
				target := info(e.Target.Owner)
				target.hierarchical += current.hierarchical
				target.random += current.random
				target.connected += current.connected + 1
			}
		}
	}
	all := toRandom + toHierarchical
	return all == 0 || float64(toRandom-toHierarchical)/float64(all) >= hierarchicalSweepiness
}

// fewerThanTwoPorts tells that parent has no order of ports to pass through
// (fewerThanTwoInOutEdges, LayerSweepTypeDecider.java line 177).
func fewerThanTwoPorts(parent *Node) bool {
	return len(portsOn(parent, sideEast)) < 2 && len(portsOn(parent, sideWest)) < 2
}

// noEdgesOn tells that no edge meets n on side (hasNoEasternPorts and
// hasNoWesternPorts, LayerSweepTypeDecider.java lines 194 and 199).
func noEdgesOn(n *Node, side PortSide) bool {
	for _, p := range portsOn(n, side) {
		if len(p.In)+len(p.Out) > 0 {
			return false
		}
	}
	return true
}
