// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"cmp"
	"slices"
)

// assignPortSides puts ports with incoming edges on the west side of their
// node and the others on the east (intermediate/PortSideProcessor.java lines
// 52 and 99). Dummy nodes, and subgraphs with external ports, have their sides
// already.
func assignPortSides(g *Graph) {
	for _, l := range g.Layers {
		for _, n := range l.Nodes {
			if n.Type != typeNormal || n.FixedPorts {
				continue
			}
			for _, p := range n.Ports {
				p.Side = sideWest
				if len(p.In) < len(p.Out) {
					p.Side = sideEast
				}
			}
		}
	}
}

// sortPortLists groups each node's ports by side
// (intermediate/PortListSorter.java line 61).
func sortPortLists(g *Graph) {
	for _, l := range g.Layers {
		for _, n := range l.Nodes {
			slices.SortStableFunc(n.Ports, func(a, b *Port) int { return cmp.Compare(a.Side, b.Side) })
			reverseWest(n.Ports)
		}
	}
}

// reverseWest turns the west ports around, as west sides run bottom to top
// (reverseWestAndSouthSide, intermediate/PortListSorter.java line 92). It keeps
// a slip of ELK's: findPortSideRange (line 110) reads the wrong index, so the
// range always ends one short of the last port, and reverse (line 139) leaves
// ranges of two or fewer alone.
func reverseWest(ports []*Port) {
	low := slices.IndexFunc(ports, func(p *Port) bool { return p.Side == sideWest })
	high := len(ports) - 1
	if low >= 0 && high > low+2 {
		slices.Reverse(ports[low:high])
	}
}
