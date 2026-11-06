// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

// Package elk lays out graphs as ELK Layered v0.12.1 lays them out for
// Mermaid v12.3.0, whose default layout it is. Paths in comments are relative
// to org.eclipse.elk.alg.layered/src/org/eclipse/elk/alg/layered.
//
// ELK reads a graph with many options; Mermaid's default configuration fixes
// nearly all of them, so here they are the code's shape rather than options.
// The one left is the direction.
//
// The steps so far, in the order they run:
//
//	importGraph       Input -> Graph                         import.go
//	toLTR             any direction -> left to right         direction.go
//	hideSelfLoops     self loops off the graph               selfloops.go
//	breakCycles       reverse the edges closing a cycle      cycles.go
//	insertLabelDummies a dummy node per label                labels.go
//	assignLayers      Layerless -> Layers                    layering.go
//	splitLongEdges    a dummy node per layer crossed         longedge.go
//	assignPortSides   incoming west, outgoing east           portorder.go
//	sortPortLists     ports grouped by side                  portorder.go
//	sortByModelOrder  layers and ports as the input has them modelorder.go
//	minimizeCrossings an order of layers with few crossings  sweep.go
//	restoreLoopPorts  self loops' ports back, on the north   selfloops.go
//	positionPorts     ports along their sides                portpos.go
//	nodeMargins       room for ports outside their node      portpos.go
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
//	fromLTR           left to right -> the input's direction direction.go
//	export            Graph -> Result                        export.go
//
// The other steps are still to come, and nothing calls this package yet. ELK
// also runs three layer constraint processors here. Mermaid sets no layer
// constraints, so they do nothing and are left out. The same goes for
// InLayerConstraintProcessor, and for the parts of crossing minimization that
// serve in-layer edges, north and south ports, several edges on a port and
// constraints between nodes. ELK sizes nodes around their labels in a system of
// cells, but Mermaid's nodes come sized, with no labels of their own, so only
// the placing of ports remains. Node placement is the balanced layout with the
// one edge straightening Mermaid gets: fixed alignments, edge priorities and
// what serves in-layer edges are left out. Edge routing runs west to east only
// and leaves out junction points. Only inline center labels are taken: the head
// and tail labels Mermaid sends for a class diagram's multiplicities are not yet.
// Self loops all stay on the north side, so what ELK does for their other sides
// and shapes is left out too.
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
// several ends, and ELK's general property table. Subgraphs and node labels
// are not done yet.
package elk
