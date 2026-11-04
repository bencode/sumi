// Package dagre lays out directed graphs in ranks, as dagre-d3-es 7.0.14,
// the dagre Mermaid v12.3.0 uses, lays them out for Mermaid: the same
// steps, in the same order, coming to the same numbers. Flowcharts, state,
// class and ER diagrams go through it.
//
// So far it does dagre's first steps (layout.js lines 26-36):
//
//	build       Mermaid's graph -> the graph to lay out    layout.go
//	acyclic     turn around the edges that close cycles    acyclic.go
//	nest        a root, and border nodes for subgraphs     nesting.go
//	rank        network simplex                            rank.go, simplex.go
//	then        label ranks, empty ranks removed, ranks from 0
//	                                                       layout.go
//
// Like dagre, the steps change one graph in place: graph, a port of
// graphlib's Graph (graph.go, edges.go). Where dagre picks among equals it
// takes the first, so the order graphlib lists nodes and edges in decides
// the layout. graphlib keeps them in JS objects, which list array indices
// first by value and the rest in the order they were added; keys does the
// same.
//
// What Mermaid never asks of dagre is left out: the greedy acyclicer, the
// tight-tree and longest-path rankers, self-loops (Mermaid splits them up
// before dagre sees them) and edge labels to the left or right (Mermaid
// centers them all).
package dagre
