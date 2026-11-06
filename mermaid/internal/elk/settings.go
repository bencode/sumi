// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

// settings is what Mermaid's options come to for one graph. Mermaid gives the
// root and subgraphs different ones, and each graph reads its own: a node's
// spacing is that of the graph it sits in, not of the graph it holds
// (getIndividualOrInherited, core util/IndividualSpacings.java line 91).
type settings struct {
	nodeNode, edgeNode, edgeEdge               float64 // spacing within a layer
	nodeLayers, edgeNodeLayers, edgeEdgeLayers float64 // spacing between layers
	portPort, portsSurrounding, labelPort      float64 // labelPort is labelPortVertical
	selfLoop, edgeLabel, component             float64
	padding                                    float64
	modelOrder                                 bool // considerModelOrder.strategy is set
	keepBends                                  bool // unnecessaryBendpoints
}

// spaced gives what a spacing option comes to at a base value when nothing sets
// it: its default in proportion to that of spacing.nodeNode, which is 20. Variables
// keep Go from working this out exactly, which ELK's doubles do not
// (ElkSpacings.java line 284).
func spaced(defaultValue, base float64) float64 { return defaultValue / 20 * base }

// spacedSettings gives the options that follow the base value alone.
func spacedSettings(base float64) settings {
	return settings{
		nodeNode: spaced(20, base), edgeNode: spaced(10, base), edgeEdge: spaced(10, base),
		nodeLayers: spaced(20, base), edgeNodeLayers: spaced(10, base), edgeEdgeLayers: spaced(10, base),
		portPort: spaced(10, base), labelPort: spaced(1, base),
		selfLoop: spaced(10, base), edgeLabel: spaced(2, base), component: spaced(20, base),
		portsSurrounding: 12,
	}
}

// rootSettings holds Mermaid's options on the root: a base value of 40, and the
// padding ELK Layered defaults to (Layered.melk line 121).
func rootSettings() settings {
	s := spacedSettings(40)
	s.padding = 12
	s.modelOrder, s.keepBends = true, true
	return s
}

// subgraphSettings holds Mermaid's options on a subgraph, which sets a base value
// of 24 and three spacings of its own (layout-elk/render.ts).
func subgraphSettings() settings {
	s := spacedSettings(24)
	s.nodeNode, s.edgeEdge, s.edgeNodeLayers = 50, 20, 30
	s.padding = 24
	return s
}
