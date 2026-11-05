// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

// Package elk lays out graphs as ELK Layered v0.12.1 lays them out for
// Mermaid v12.3.0, whose default layout it is. Paths in comments are relative
// to org.eclipse.elk.alg.layered/src/org/eclipse/elk/alg/layered.
//
// ELK reads a graph with many options; Mermaid's default configuration fixes
// nearly all of them, so here they are the code's shape rather than options.
// The one left is the direction.
//
// So far a graph only goes in and comes back out:
//
//	importGraph  Input -> Graph                         import.go
//	toLTR        any direction -> left to right         direction.go
//	fromLTR      left to right -> the input's direction direction.go
//	export       Graph -> Result                        export.go
//
// Everything between toLTR and fromLTR is still to come, and nothing calls
// this package yet.
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
