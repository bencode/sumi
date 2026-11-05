// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"slices"
	"testing"
)

// The values expected below come from elkjs 0.12.1.

// placed runs positionPorts and nodeMargins on one node of the given size with
// east, west and north ports.
func placed(size Vec, east, west, north int) *Node {
	n := &Node{Type: typeNormal, Size: size}
	for _, side := range []struct {
		side  PortSide
		count int
	}{{sideNorth, north}, {sideEast, east}, {sideWest, west}} {
		for range side.count {
			n.Ports = append(n.Ports, &Port{Owner: n, Side: side.side})
		}
	}
	g := &Graph{Layers: []*Layer{{Nodes: []*Node{n}}}}
	positionPorts(g)
	nodeMargins(g)
	return n
}

func coordinates(n *Node, side PortSide, of func(Vec) float64) []float64 {
	var all []float64
	for _, p := range portsOn(n, side) {
		all = append(all, of(p.Pos))
	}
	return all
}

func TestPortPositions(t *testing.T) {
	x := func(v Vec) float64 { return v.X }
	y := func(v Vec) float64 { return v.Y }
	tests := []struct {
		height float64
		east   []float64
	}{
		{45, []float64{22.5}},
		{45, []float64{18.333333333333336, 26.66666666666667}},
		{45, []float64{15, 20, 25, 30}},
		{152, []float64{
			28.857142857142858, 47.714285714285715, 66.57142857142857,
			85.42857142857143, 104.28571428571429, 123.14285714285715,
		}},
		{14, []float64{10}},
		{14, []float64{8, 6}},
		{14, []float64{8.8, 7.600000000000001, 6.400000000000002, 5.200000000000003}},
		{10, []float64{6.666666666666668, 3.3333333333333357}},
		{5, []float64{10}},
		{5, []float64{5, 0}},
	}
	for _, tt := range tests {
		count := len(tt.east)
		n := placed(Vec{60, tt.height}, count, count, 0)
		if got := coordinates(n, sideEast, y); !slices.Equal(got, tt.east) {
			t.Errorf("height %v: east %v, want %v", tt.height, got, tt.east)
		}
		west := slices.Clone(tt.east)
		slices.Reverse(west)
		if got := coordinates(n, sideWest, y); !slices.Equal(got, west) {
			t.Errorf("height %v: west %v, want %v", tt.height, got, west)
		}
		if got := coordinates(n, sideEast, x)[0]; got != 60 {
			t.Errorf("height %v: east x %v", tt.height, got)
		}
		if got := coordinates(n, sideWest, x)[0]; got != 0 {
			t.Errorf("height %v: west x %v", tt.height, got)
		}
	}

	north := []struct {
		width float64
		want  []float64
	}{
		{152, []float64{54.66666666666667, 97.33333333333334}},
		{152, []float64{37.6, 63.2, 88.80000000000001, 114.4}},
		{10, []float64{7.333333333333332, 2.6666666666666643}},
	}
	for _, tt := range north {
		n := placed(Vec{tt.width, 45}, 0, 0, len(tt.want))
		if got := coordinates(n, sideNorth, x); !slices.Equal(got, tt.want) {
			t.Errorf("width %v: north %v, want %v", tt.width, got, tt.want)
		}
	}
}

func TestDummyPortsStay(t *testing.T) {
	n := &Node{Type: typeLongEdge, Size: Vec{0, 5}}
	n.Ports = []*Port{{Owner: n, Side: sideEast, Pos: Vec{7, 7}}}
	g := &Graph{Layers: []*Layer{{Nodes: []*Node{n}}}}
	positionPorts(g)
	if n.Ports[0].Pos != (Vec{7, 7}) {
		t.Errorf("port at %v", n.Ports[0].Pos)
	}
}

func TestNodeMargins(t *testing.T) {
	if got, want := placed(Vec{60, 5}, 1, 0, 0).Margin, (Spacing{Bottom: 5}); got != want {
		t.Errorf("short node: %v, want %v", got, want)
	}
	want := Spacing{Right: 3.1999999999999993, Left: 3.200000000000003}
	if got := placed(Vec{5, 45}, 0, 0, 4).Margin; got != want {
		t.Errorf("narrow node: %v, want %v", got, want)
	}

	// Mirrored from the left, a node is not at 0, which rounds differently.
	n := &Node{Type: typeNormal, Pos: Vec{29, 0}, Size: Vec{1, 5}}
	for _, at := range []Vec{{4.333333333333332, 0}, {-3.3333333333333357, 0}, {0, 0}, {0, 5}} {
		n.Ports = append(n.Ports, &Port{Owner: n, Pos: at})
	}
	nodeMargins(&Graph{Layers: []*Layer{{Nodes: []*Node{n}}}})
	want = Spacing{Right: 3.3333333333333286, Left: 3.3333333333333357}
	if n.Margin != want {
		t.Errorf("node at 29: %v, want %v", n.Margin, want)
	}
}
