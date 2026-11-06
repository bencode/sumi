// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

// portsToNode gives the ports of g's node the places their dummy nodes reached,
// for g's parent to route to (graphLayoutToNode,
// intermediate/HierarchicalNodeResizingProcessor.java line 79). The dummies move out
// to where the ports are, past the border by their offset
// (getExternalPortPosition, graph/LGraphUtil.java line 980). That leaves nothing for
// intermediate/HierarchicalPortOrthogonalEdgeRouter.java to do: it serves north and
// south ports and fixed port positions, and no other edge ends slanted.
func portsToNode(g *Graph) {
	p := g.Padding
	for _, n := range g.Layerless {
		if n.Type != typeExternalPort {
			continue
		}
		pos := Vec{Y: n.Pos.Y + (p.Top + g.Offset.Y)}
		if n.ExtSide == sideEast {
			pos.X = g.Size.X + p.Left + p.Right + n.BorderOffset
			n.Pos.X = g.Size.X + p.Right + n.BorderOffset - g.Offset.X
		} else {
			pos.X = -n.BorderOffset
			n.Pos.X = -(p.Left + n.BorderOffset + g.Offset.X)
		}
		n.Origin.Pos = pos
	}
}
