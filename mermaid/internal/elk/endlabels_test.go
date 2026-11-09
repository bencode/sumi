// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"math"
	"testing"
)

// The cases come from what Mermaid 12.3.0 gave elkjs 0.12.1 for class diagrams.
func TestEndLabels(t *testing.T) {
	node := func(id string) InputNode { return InputNode{ID: id, Width: 34.125, Height: 81} }
	narrow, short, wide := &Size{6.125, 16.5}, &Size{18.125, 28.5}, &Size{38.90625, 28.5}
	tests := []struct {
		name       string
		nodes      []InputNode
		edges      []InputEdge
		tails      []Vec
		heads      []Vec
		lastNodeAt Vec
	}{
		{
			name:       "the labels push the nodes apart",
			nodes:      []InputNode{node("A"), node("B")},
			edges:      []InputEdge{{ID: "e", Source: "A", Target: "B", Tail: narrow, Head: wide}},
			tails:      []Vec{{34.0625, 97}},
			heads:      []Vec{{34.0625, 153.5}},
			lastNodeAt: Vec{12, 186},
		},
		{
			name:  "an edge reversed in a cycle keeps its labels at their ends",
			nodes: []InputNode{node("A"), node("B")},
			edges: []InputEdge{
				{ID: "e0", Source: "A", Target: "B", Tail: narrow, Head: short},
				{ID: "e1", Source: "B", Target: "A", Tail: narrow, Head: short},
			},
			tails:      []Vec{{31.708333333333336, 97}, {36.41666666666667, 182}},
			heads:      []Vec{{31.708333333333336, 198.5}, {36.41666666666667, 113.5}},
			lastNodeAt: Vec{12, 231},
		},
		{
			name:  "labels side by side step out of each other's way",
			nodes: []InputNode{node("A"), node("B"), node("C"), node("D")},
			edges: []InputEdge{
				{ID: "e0", Source: "A", Target: "B", Tail: narrow, Head: short},
				{ID: "e1", Source: "A", Target: "C", Tail: narrow, Head: short},
				{ID: "e2", Source: "A", Target: "D", Tail: narrow, Head: short},
			},
			tails:      []Vec{{110.71875, 97}, {114.25, 113.5}, {117.78125, 97}},
			heads:      []Vec{{34.0625, 170}, {114.25, 170}, {194.4375, 170}},
			lastNodeAt: Vec{172.375, 202.5},
		},
	}
	near := func(l *PlacedLabel, want Vec) bool {
		return l != nil && math.Abs(l.X-want.X) < 1e-9 && math.Abs(l.Y-want.Y) < 1e-9
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Layout(Input{Direction: Down, Nodes: tt.nodes, Edges: tt.edges})
			for i, e := range r.Edges {
				if !near(e.Tail, tt.tails[i]) || !near(e.Head, tt.heads[i]) {
					t.Errorf("edge %d: tail %v head %v, want %v %v", i, e.Tail, e.Head, tt.tails[i], tt.heads[i])
				}
			}
			last := r.Nodes[len(r.Nodes)-1]
			if last.X != tt.lastNodeAt.X || last.Y != tt.lastNodeAt.Y {
				t.Errorf("last node at %v,%v, want %v", last.X, last.Y, tt.lastNodeAt)
			}
		})
	}
}
