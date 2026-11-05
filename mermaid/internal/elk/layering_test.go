// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// graphOf builds a left to right graph from edges written "a>b b>c". Nodes come
// in the order first named, and edges are called e0, e1 and so on.
func graphOf(spec string) *Graph {
	in := Input{Direction: Right}
	seen := map[string]bool{}
	for i, pair := range strings.Fields(spec) {
		source, target, _ := strings.Cut(pair, ">")
		for _, id := range []string{source, target} {
			if !seen[id] {
				seen[id] = true
				in.Nodes = append(in.Nodes, InputNode{ID: id})
			}
		}
		in.Edges = append(in.Edges, InputEdge{ID: fmt.Sprint("e", i), Source: source, Target: target})
	}
	return importGraph(in)
}

// layerNames writes g's layers like "a | b,. | c", a dot for a dummy node.
func layerNames(g *Graph) string {
	layers := make([]string, len(g.Layers))
	for i, l := range g.Layers {
		names := make([]string, len(l.Nodes))
		for j, n := range l.Nodes {
			names[j] = n.ID
			if n.Type == typeLongEdge {
				names[j] = "."
			}
		}
		layers[i] = strings.Join(names, ",")
	}
	return strings.Join(layers, " | ")
}

func edgeNamed(g *Graph, id string) *Edge {
	all := edges(g)
	return all[slices.IndexFunc(all, func(e *Edge) bool { return e.ID == id })]
}

// forwardSpec writes a random tree on n nodes plus up to extra more edges, all
// going forward. Same arguments, same graph.
func forwardSpec(n, extra int) string {
	seed := uint32(1)
	next := func(k int) int {
		seed = seed*1664525 + 1013904223
		return int(seed>>8) % k
	}
	var pairs []string
	for i := 1; i < n; i++ {
		pairs = append(pairs, fmt.Sprintf("n%d>n%d", next(i), i))
	}
	for range extra {
		if a, b := next(n), next(n); a != b {
			pairs = append(pairs, fmt.Sprintf("n%d>n%d", min(a, b), max(a, b)))
		}
	}
	return strings.Join(pairs, " ")
}

// The reversed edges and layers expected below come from elkjs 0.12.1.

func TestBreakCycles(t *testing.T) {
	tests := []struct {
		spec     string
		reversed []string
	}{
		{"a>b b>c c>a", []string{"e2"}},
		{"a>b b>a c>a", []string{"e1"}},
		{"a>b c>b b>a", []string{"e0"}},
		{"a>b a>c b>d c>d a>d", nil},
	}
	for _, tt := range tests {
		g := graphOf(tt.spec)
		breakCycles(g)
		var reversed []string
		for _, e := range edges(g) {
			if e.Reversed {
				reversed = append(reversed, e.ID)
			}
		}
		if !slices.Equal(reversed, tt.reversed) {
			t.Errorf("%s: reversed %v, want %v", tt.spec, reversed, tt.reversed)
		}
	}
}

func TestLayers(t *testing.T) {
	tests := []struct{ spec, layered, split string }{
		{"a>b b>c c>a", "a | b | c", "a | b,. | c"},
		{"a>b a>c b>d c>d a>d", "a | b,c | d", "a | b,c,. | d"},
		{"a>b c>d e>f f>g", "e,a,c | f,b,d | g", "e,a,c | f,b,d | g"},
		{"a>c b>c c>d c>e d>f e>f f>g d>g", "a,b | c | d,e | f | g", "a,b | c | d,e | f,. | g"},
		{"a>b a>c b>d a>c c>d", "a | b,c | d", "a | b,c | d"},
		{"j>d e>g d>h c>h c>g a>h g>i b>e a>i a>h f>a", "j,b,f | d,c,e,a | h,g | i", "j,b,f | d,c,e,a | h,g,. | i"},
		// Components: equal in size, then not sorted by size.
		{"a>b a>c d>e e>f", "a,d | b,c,e | f", "a,d | b,c,e | f"},
		{"f>g b>e g>c a>d h>d", "f,b,a,h | g,e,d | c", "f,b,a,h | g,e,d | c"},
		// Balance: moved, held back by ELK's minimalSpan, and sharing layers
		// with an earlier component.
		{"c>k e>i a>e c>a k>i", "c | a | k,e | i", "c | a,. | k,e | i"},
		{"a>b b>c b>d a>e e>f d>f d>e c>f b>c c>f", "a | b | c,d | e | f", "a | b,. | c,d,. | e,.,.,. | f"},
		{"a>b a>c c>d c>e b>e d>e f>g f>h h>i g>j j>k h>l j>m g>i f>g", "f,a | g,h,b,c | j,i,l,d | k,m,e", "f,a | g,h,b,c | j,i,l,d,.,. | k,m,e"},
	}
	for _, tt := range tests {
		g := graphOf(tt.spec)
		breakCycles(g)
		assignLayers(g)
		if got := layerNames(g); got != tt.layered {
			t.Errorf("%s: layers %q, want %q", tt.spec, got, tt.layered)
		}
		splitLongEdges(g)
		if got := layerNames(g); got != tt.split {
			t.Errorf("%s: split %q, want %q", tt.spec, got, tt.split)
		}
	}
}

// From 40 nodes up, subtrees come off first, and nodes change places.
func TestSubtrees(t *testing.T) {
	g := graphOf(forwardSpec(40, 10))
	assignLayers(g)
	want := "n0 | n1,n18 | n2,n24,n6 | n3,n33,n21,n5,n4,n7 | n17,n11,n14,n8,n39,n34,n37,n20,n32,n13 | " +
		"n19,n9,n26,n12,n15,n16,n10,n27,n36 | n38,n22,n29,n23,n25,n30 | n28,n31 | n35"
	if got := layerNames(g); got != want {
		t.Errorf("layers %q, want %q", got, want)
	}
}

// This graph needs more exchanges than the limit allows, so its layers are not
// optimal, in just the way ELK's are not.
func TestIterationLimit(t *testing.T) {
	g := graphOf(forwardSpec(700, 1400))
	assignLayers(g)
	layerOf := map[*Node]int{}
	for i, l := range g.Layers {
		for _, n := range l.Nodes {
			layerOf[n] = i
		}
	}
	total := 0
	for _, e := range edges(g) {
		total += layerOf[e.Target.Owner] - layerOf[e.Source.Owner]
	}
	if len(g.Layers) != 21 || total != 6784 {
		t.Errorf("%d layers, total length %d, want 21 and 6784", len(g.Layers), total)
	}
}

func TestLongEdgeRoundTrip(t *testing.T) {
	g := graphOf("a>b b>c c>d a>d d>a")
	breakCycles(g)
	assignLayers(g)
	splitLongEdges(g)
	for i, l := range g.Layers {
		for j, n := range l.Nodes {
			n.Pos = Vec{float64(i * 10), float64(j)}
		}
	}
	for _, e := range edges(g) {
		if e.ID == "e4" {
			if !e.Reversed || e.ModelOrder != 4 {
				t.Errorf("a part of e4 has reversed %v, order %d", e.Reversed, e.ModelOrder)
			}
			e.Bends = []Vec{{e.Source.Owner.Pos.X + 1, e.Source.Owner.Pos.Y}}
		}
	}
	dummy := edgeNamed(g, "e4").Target.Owner
	if dummy.Size != (Vec{0, 1}) || !slices.Equal(sides(dummy), []PortSide{sideWest, sideEast}) {
		t.Errorf("dummy size %v, sides %v", dummy.Size, sides(dummy))
	}
	joinLongEdges(g)
	if got, want := layerNames(g), "a | b | c | d"; got != want {
		t.Errorf("layers %q, want %q", got, want)
	}
	e := edgeNamed(g, "e4")
	want := []Vec{{1, 0}, {10, 2}, {11, 2}, {20, 2}, {21, 2}}
	if !slices.Equal(e.Bends, want) {
		t.Errorf("joined bends %v, want %v", e.Bends, want)
	}
	restoreReversed(g)
	slices.Reverse(want)
	if e.Reversed || e.Source.Owner.ID != "d" || e.Target.Owner.ID != "a" || !slices.Equal(e.Bends, want) {
		t.Errorf("restored %s to %s reversed %v bends %v", e.Source.Owner.ID, e.Target.Owner.ID, e.Reversed, e.Bends)
	}
}
