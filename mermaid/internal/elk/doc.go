// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

// Package elk lays out graphs as ELK Layered v0.12.1 lays them out for
// Mermaid v12.3.0, whose default layout it is. Paths in comments are relative
// to org.eclipse.elk.alg.layered/src/org/eclipse/elk/alg/layered.
//
// ELK reads a graph with many options; Mermaid's default configuration fixes
// nearly all of them, so here they are the code's shape rather than options.
// What is left is the direction and hierarchy of each graph, the sizes of titles
// and the settings of settings.go, one set for the root and one for subgraphs.
//
// Layout takes a tree of nodes. A subgraph is laid out one of two ways, by its
// Hierarchy (layout.go). Separate ones go first, on their own, and their parent
// sees a node of fixed size; such a layout splits into connected components,
// which it orders on their own and then puts in rows (components.go). Included
// ones run alongside their parent, which never splits: first each graph bottom up
// to the end of the steps before ordering, then crossing minimization over them
// all, then each graph bottom up to the end. An edge across an included subgraph's
// border is cut there into one segment per graph, with a dummy node standing for the
// place it crosses and a port on the subgraph (cross.go). The segments are laid out
// like any edge and joined up at the end (cross_stitch.go).
//
// The steps of one graph, in the order they run:
//
//	toLTR             any direction -> left to right         direction.go
//	reverseConstrainedEdges edges to fit external ports      constraints.go
//	hideSelfLoops     self loops off the graph               selfloops.go
//	breakCycles       reverse the edges closing a cycle      cycles.go
//	insertLabelDummies a dummy node per label                labels.go
//	hideSeparateNodes external port dummies off the graph    constraints.go
//	assignLayers      Layerless -> Layers                    layering.go
//	placeConstrainedNodes those dummies in layers of their own constraints.go
//	splitLongEdges    a dummy node per layer crossed         longedge.go
//	assignPortSides   incoming west, outgoing east           portorder.go
//	invertPorts       a dummy node per port the wrong way    inverted.go
//	sortPortLists     ports grouped by side                  portorder.go
//	sortByModelOrder  layers and ports as the input has them modelorder.go
//	minimizeCrossings an order of layers with few crossings  sweep.go
//	greedySwitch      neighbors swapped, components only     greedy.go
//	restoreLoopPorts  self loops' ports back, on the north   selfloops.go
//	positionPorts     ports along their sides, and titles    portpos.go
//	nodeMargins       room for what sticks out of a node     portpos.go
//	routeSelfLoops    bends and labels of self loops         selfloops.go
//	switchLabelDummies label dummies to the middle of edges  labels.go
//	sizeLabelDummies  label dummies down to the label        labels.go
//	placeNodes        the y of each node                     bk.go
//	measureLayers     widths of layers, height of the graph  route.go
//	routeEdges        the x of each node, the bends of edges route.go
//	joinLongEdges     the dummy nodes out again              longedge.go
//	restoreSelfLoops  self loops back on the graph           selfloops.go
//	removeLabelDummies labels placed, their dummies out      labels.go
//	restoreReversed   reversed edges back                    cycles.go
//	collectNodes      all nodes back in Layerless            route.go
//	fitGraph          the graph's minimum size               layout.go
//	portsToNode       ports placed on the subgraph's node    external.go
//	fromLTR           left to right -> the input's direction direction.go
//
// Around them run importGraph (import.go), Input -> Graph, and export (export.go),
// Graph -> Result. In a layout of components, fitGraph comes last, after they
// are back together.
//
// Nothing calls this package yet. Mermaid sets no layer constraints, so only those
// of external port dummies exist, and InLayerConstraintProcessor does nothing and is
// left out. Edges within a layer come only from inverted ports, the ports of a
// subgraph that an edge reaches from the wrong side. Crossing minimization leaves
// out what serves north and south ports, several edges on a port and constraints
// between nodes, and works bottom up: ELK's sweeps into subgraphs are not done
// yet, so graphs that take them still differ. ELK sizes nodes around their labels in
// a system of cells, but Mermaid's nodes come sized, with no labels of their own,
// so only the placing of ports and titles remains. Node placement is the balanced
// layout with the one edge straightening Mermaid gets: fixed alignments and edge
// priorities are left out. Edge routing runs west to east only and leaves out
// junction points. Only inline center labels are taken: the head and tail labels
// Mermaid sends for a class diagram's multiplicities are not yet. Self loops all
// stay on the north side, so what ELK does for their other sides and shapes is
// left out too. A self loop on a subgraph with external ports has its ports on
// opposite sides, as in ELK, but runs over the top without ELK's room at the sides,
// so that loop and the space around the subgraph differ from ELK.
//
// Layout has to match elkjs, which is ELK compiled to JavaScript, not ELK on a
// JVM: a float there is a double, and ELK's random numbers are seeded the same
// way each time.
//
// In a Graph, edges join ports, not nodes: each edge gets a port of its own at
// both ends. That is what spreads edges along a node's side. Steps change one
// Graph in place, as ELK's processors do: they add and remove nodes, and move
// ports between them, and tell them apart by identity.
//
// What Mermaid's input never holds is left out: ports of its own, edges with
// several ends, and ELK's general property table. An edge across the border of a
// subgraph laid out on its own panics, as ELK refuses it too. A subgraph's own
// algorithm (@{algorithm}) and the labels of leaf nodes are not done yet.
package elk
