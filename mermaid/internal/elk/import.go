// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "fmt"

// defaultPadding is ELK Layered's padding; Mermaid sets none on the root
// (Layered.melk line 121).
const defaultPadding = 12

// importGraph turns in into a Graph, all nodes first, as
// importHierarchicalGraph does (graph/transform/ElkGraphImporter.java lines
// 96, 309).
func importGraph(in Input) *Graph {
	g := &Graph{
		Direction: in.Direction,
		Padding:   Spacing{defaultPadding, defaultPadding, defaultPadding, defaultPadding},
	}
	byID := make(map[string]*Node, len(in.Nodes))
	for i, n := range in.Nodes {
		node := &Node{ID: n.ID, Size: Vec{n.Width, n.Height}, ModelOrder: i}
		g.Layerless = append(g.Layerless, node)
		byID[n.ID] = node
	}
	for i, e := range in.Edges {
		importEdge(g, byID, e, i)
	}
	return g
}

// importEdge adds e with a new port at each end, the source's first
// (transformEdge, ElkGraphImporter.java line 1084). An unknown end panics, as
// ELK throws: the parser declares every node an edge names.
func importEdge(g *Graph, byID map[string]*Node, e InputEdge, order int) {
	source, target := byID[e.Source], byID[e.Target]
	if source == nil || target == nil {
		panic(fmt.Sprintf("elk: edge %s has an unknown end", e.ID))
	}
	side := sideOf[g.Direction]
	edge := &Edge{ID: e.ID, ModelOrder: order}
	edge.Source, edge.Target = newPort(source, side), newPort(target, opposed[side])
	edge.Source.Out = append(edge.Source.Out, edge)
	edge.Target.In = append(edge.Target.In, edge)
	g.SelfLoops = g.SelfLoops || source == target
	if e.Label != nil {
		edge.Labels = []*Label{{Size: Vec{e.Label.Width, e.Label.Height}}}
		g.CenterLabels = true
	}
}

// newPort adds a port on one side of n. Ports are never shared, since Mermaid
// sets mergeEdges to false (createPort, graph/LGraphUtil.java line 433).
func newPort(n *Node, side PortSide) *Port {
	p := &Port{Owner: n, Side: side}
	n.Ports = append(n.Ports, p)
	return p
}
