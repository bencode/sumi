// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "slices"

// positionPorts spreads the ports of real nodes along their sides, top to
// bottom and left to right, and places their titles
// (intermediate/LabelAndNodeSizeProcessor.java line 65). A node's ports run
// clockwise, so west ports go in reverse. ELK takes labelPortVertical, as the
// height of the empty cells above and below, off the top and bottom of
// portsSurrounding (updateVerticalInsidePortLabelCellPadding,
// nodespacing/internal/algorithm/CellSystemConfigurator.java line 142). A subgraph
// with external ports has them in place already, by portsToNode.
func positionPorts(g *Graph) {
	s := g.Settings
	endPadding := s.portsSurrounding - s.labelPort
	for _, n := range nodes(g) {
		if n.Type != typeNormal {
			continue
		}
		if !n.FixedPorts {
			spreadPorts(n, s, endPadding)
		}
		if n.Title != nil {
			n.Title.Pos = titlePosition(n, g.Direction)
		}
	}
}

// titlePosition places n's title in the graph turned left to right, which turns
// Mermaid's top center too: going down it is left center, and going up right
// center (applyHorizontalModeLabelLayout and applyVerticalModeLabelLayout,
// nodespacing/cellsystem/LabelCell.java lines 241 and 286; transposeNodeLabelPlacement,
// intermediate/GraphTransformer.java line 683). The title's cell takes titleGap
// off each side of the node (applyWidthsToRow,
// nodespacing/cellsystem/GridContainerCell.java line 411).
func titlePosition(n *Node, d Direction) Vec {
	size := n.Title.Size
	center := func(length, title float64) float64 {
		return titleGap + (length-titleGap-titleGap-title)/2
	}
	switch d {
	case Down:
		return Vec{titleGap, center(n.Size.Y, size.Y)}
	case Up:
		return Vec{n.Size.X - titleGap - size.X, center(n.Size.Y, size.Y)}
	}
	return Vec{center(n.Size.X, size.X), titleGap}
}

func spreadPorts(n *Node, s settings, endPadding float64) {
	east, west, north := portsOn(n, sideEast), portsOn(n, sideWest), portsOn(n, sideNorth)
	slices.Reverse(west)
	for i, y := range spread(n.Size.Y, len(east), endPadding, s.portPort) {
		east[i].Pos = Vec{n.Size.X, y}
	}
	for i, y := range spread(n.Size.Y, len(west), endPadding, s.portPort) {
		west[i].Pos = Vec{0, y}
	}
	for i, x := range spread(n.Size.X, len(north), s.portsSurrounding, s.portPort) {
		north[i].Pos = Vec{x, 0}
	}
}

func portsOn(n *Node, side PortSide) []*Port {
	var ports []*Port
	for _, p := range n.Ports {
		if p.Side == side {
			ports = append(ports, p)
		}
	}
	return ports
}

// spread gives count positions along a side, spaced evenly with the room left
// inside padding (placeVerticalFreePorts and placeHorizontalFreePorts,
// nodespacing/internal/algorithm/PortPlacementCalculator.java lines 285 and
// 110). Where there is too little room ports move closer, and with less than
// none they run backwards. A single port goes to the middle, but with less than
// no room it stays at the padding: ELK then takes the branch that divides by
// count-1 and only uses the result between ports.
func spread(length float64, count int, padding, portSpacing float64) []float64 {
	room := length - padding - padding
	if count == 1 {
		return []float64{padding + max(room, 0)/2}
	}
	gap := portSpacing + (room-portSpacing*float64(count+1))/float64(count+1)
	all := make([]float64, count)
	position := padding
	for i := range all {
		position += gap
		all[i] = position
	}
	return all
}

// nodeMargins makes room for titles and ports outside their node, as on nodes too
// small for the padding (intermediate/InnermostNodeMarginCalculator.java line 57 and
// processNode, nodespacing/NodeMarginCalculator.java line 138). The box grows
// as ElkRectangle.union grows it (core/math/ElkRectangle.java line 151).
func nodeMargins(g *Graph) {
	for _, n := range nodes(g) {
		x, y, width, height := n.Pos.X, n.Pos.Y, n.Size.X, n.Size.Y
		union := func(at, size Vec) {
			left, top := min(x, at.X), min(y, at.Y)
			right, bottom := max(x+width, at.X+size.X), max(y+height, at.Y+size.Y)
			x, y, width, height = left, top, right-left, bottom-top
		}
		if n.Title != nil {
			union(n.Title.Pos.add(n.Pos), n.Title.Size)
		}
		for _, p := range n.Ports {
			union(p.Pos.add(n.Pos), Vec{})
		}
		n.Margin = Spacing{
			Top:    max(0, n.Pos.Y-y),
			Right:  max(0, x+width-(n.Pos.X+n.Size.X)),
			Bottom: max(0, y+height-(n.Pos.Y+n.Size.Y)),
			Left:   max(0, n.Pos.X-x),
		}
	}
}
