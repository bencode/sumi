// Package graphlayout places what flowcharts, state, class and ER diagrams draw,
// as Mermaid v12.3.0 places it with its default configuration. It is written after
// Mermaid's layout-elk, and carries Mermaid's MIT notice in LICENSE. File names in
// comments are those of layout-elk.
//
// Mermaid does more than call ELK. Before ELK it builds a tree of what diagrams
// give as a flat list, decides which subgraphs ELK lays out on their own, and
// turns round or moves edges that ELK would otherwise route badly (input.go,
// feedback.go, nesting.go). What follows ELK is not here yet, and nothing calls
// this package.
package graphlayout
