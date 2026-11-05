// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"reflect"
	"slices"
	"testing"
)

// Every expected value here comes from running elkjs 0.12.1 on sample with
// Mermaid's options, reading the graph after the step in question.

var sample = Input{
	Nodes: []InputNode{{"a", 100, 40}, {"b", 60.5, 30}, {"c", 80, 50}},
	Edges: []InputEdge{
		{ID: "e0", Source: "a", Target: "b", Label: &Size{30, 10}},
		{ID: "e1", Source: "a", Target: "c"},
		{ID: "e2", Source: "b", Target: "b"},
		{ID: "e3", Source: "c", Target: "a"},
	},
}

func withDirection(d Direction) Input {
	in := sample
	in.Direction = d
	return in
}

func sides(n *Node) []PortSide {
	all := make([]PortSide, len(n.Ports))
	for i, p := range n.Ports {
		all[i] = p.Side
	}
	return all
}

func TestImport(t *testing.T) {
	g := importGraph(withDirection(Down))
	want := [][]PortSide{
		{sideSouth, sideSouth, sideNorth},
		{sideNorth, sideSouth, sideNorth},
		{sideNorth, sideSouth},
	}
	for i, n := range g.Layerless {
		if n.ID != sample.Nodes[i].ID || n.ModelOrder != i {
			t.Errorf("node %d is %s, order %d", i, n.ID, n.ModelOrder)
		}
		if !reflect.DeepEqual(sides(n), want[i]) {
			t.Errorf("%s: ports %v, want %v", n.ID, sides(n), want[i])
		}
	}
	if want := (Spacing{12, 12, 12, 12}); g.Padding != want {
		t.Errorf("padding %v, want %v", g.Padding, want)
	}
	if !g.SelfLoops || !g.CenterLabels {
		t.Errorf("self loops %v, center labels %v", g.SelfLoops, g.CenterLabels)
	}
	plain := importGraph(Input{Nodes: sample.Nodes, Edges: sample.Edges[1:2]})
	if plain.SelfLoops || plain.CenterLabels {
		t.Errorf("plain: self loops %v, center labels %v", plain.SelfLoops, plain.CenterLabels)
	}
}

func TestToLTR(t *testing.T) {
	sidesLTR := [][]PortSide{
		{sideEast, sideEast, sideWest},
		{sideWest, sideEast, sideWest},
		{sideWest, sideEast},
	}
	tests := []struct {
		direction Direction
		pos, size []Vec
		port      []Vec // every port of a node sits at one place
		labelPos  Vec
		labelSize Vec
	}{
		{Down, []Vec{{}, {}, {}}, []Vec{{40, 100}, {30, 60.5}, {50, 80}}, []Vec{{}, {}, {}}, Vec{}, Vec{10, 30}},
		{Right, []Vec{{}, {}, {}}, []Vec{{100, 40}, {60.5, 30}, {80, 50}}, []Vec{{}, {}, {}}, Vec{}, Vec{30, 10}},
		{Up, []Vec{{10, 0}, {20, 0}, {}}, []Vec{{40, 100}, {30, 60.5}, {50, 80}}, []Vec{{40, 0}, {30, 0}, {50, 0}}, Vec{40, 0}, Vec{10, 30}},
		{Left, []Vec{{}, {39.5, 0}, {20, 0}}, []Vec{{100, 40}, {60.5, 30}, {80, 50}}, []Vec{{100, 0}, {60.5, 0}, {80, 0}}, Vec{70, 0}, Vec{30, 10}},
	}
	for _, tt := range tests {
		g := importGraph(withDirection(tt.direction))
		toLTR(g)
		for i, n := range g.Layerless {
			if n.Pos != tt.pos[i] || n.Size != tt.size[i] {
				t.Errorf("%d: %s at %v size %v, want %v %v", tt.direction, n.ID, n.Pos, n.Size, tt.pos[i], tt.size[i])
			}
			if !reflect.DeepEqual(sides(n), sidesLTR[i]) {
				t.Errorf("%d: %s ports %v, want %v", tt.direction, n.ID, sides(n), sidesLTR[i])
			}
			for _, p := range n.Ports {
				if p.Pos != tt.port[i] {
					t.Errorf("%d: %s port at %v, want %v", tt.direction, n.ID, p.Pos, tt.port[i])
				}
			}
		}
		label := g.Layerless[0].Ports[0].Out[0].Labels[0]
		if label.Pos != tt.labelPos || label.Size != tt.labelSize {
			t.Errorf("%d: label %+v, want %v %v", tt.direction, *label, tt.labelPos, tt.labelSize)
		}
	}
}

// frameNode and frameEdge hold a graph as the last step before fromLTR leaves
// it. Layout has moved ports, so ends are indexes into the node's ports.
type frameNode struct {
	id        string
	order     int
	pos, size Vec
	ports     []Vec
}

type frameEdge struct {
	id             string
	order          int
	from, fromPort int
	to, toPort     int
	bends          []Vec
	labelPos       *Vec
	labelSize      Vec
}

type frame struct {
	size, offset Vec
	nodes        []frameNode
	edges        []frameEdge
}

func (f frame) graph(d Direction) *Graph {
	g := &Graph{Direction: d, Size: f.size, Offset: f.offset, Padding: Spacing{12, 12, 12, 12}}
	for _, n := range f.nodes {
		node := &Node{ID: n.id, ModelOrder: n.order, Pos: n.pos, Size: n.size}
		for _, pos := range n.ports {
			newPort(node, sideUndefined).Pos = pos
		}
		g.Layerless = append(g.Layerless, node)
	}
	for _, e := range f.edges {
		edge := &Edge{
			ID: e.id, ModelOrder: e.order, Bends: slices.Clone(e.bends),
			Source: g.Layerless[e.from].Ports[e.fromPort],
			Target: g.Layerless[e.to].Ports[e.toPort],
		}
		if e.labelPos != nil {
			edge.Labels = []*Label{{Pos: *e.labelPos, Size: e.labelSize}}
		}
		edge.Source.Out = append(edge.Source.Out, edge)
		edge.Target.In = append(edge.Target.In, edge)
	}
	return g
}

var tall = frame{
	size: Vec{200, 185.25},
	nodes: []frameNode{
		{"a", 0, Vec{0, 85.25}, Vec{40, 100}, []Vec{{40, 30}, {40, 50}, {40, 70}}},
		{"c", 2, Vec{80, 105.25}, Vec{50, 80}, []Vec{{0, 50}, {0, 30}}},
		{"b", 1, Vec{170, 20}, Vec{30, 60.5}, []Vec{{14, 0}, {16, 0}, {0, 30.25}}},
	},
	edges: []frameEdge{
		{"e0", 0, 0, 0, 2, 2, []Vec{{60, 115.25}, {60, 50.25}}, &Vec{100, 35.25}, Vec{10, 30}},
		{"e1", 1, 0, 1, 1, 1, nil, nil, Vec{}},
		{"e2", 2, 2, 0, 2, 1, []Vec{{184, 0}, {186, 0}}, nil, Vec{}},
		{"e3", 3, 1, 0, 0, 2, nil, nil, Vec{}},
	},
}

var wide = frame{
	size: Vec{340.5, 130.5},
	nodes: []frameNode{
		{"a", 0, Vec{0, 83}, Vec{100, 40}, []Vec{{100, 15}, {100, 20}, {100, 25}}},
		{"c", 2, Vec{160, 80.5}, Vec{80, 50}, []Vec{{0, 30}, {0, 20}}},
		{"b", 1, Vec{280, 20}, Vec{60.5, 30}, []Vec{{24.166666666666668, 0}, {36.333333333333336, 0}, {0, 15}}},
	},
	edges: []frameEdge{
		{"e0", 0, 0, 0, 2, 2, []Vec{{120, 98}, {120, 35}}, &Vec{185, 29.5}, Vec{30, 10}},
		{"e1", 1, 0, 1, 1, 1, []Vec{{140, 103}, {140, 100.5}}, nil, Vec{}},
		{"e2", 2, 2, 0, 2, 1, []Vec{{304.1666666666667, 0}, {316.3333333333333, 0}}, nil, Vec{}},
		{"e3", 3, 1, 0, 0, 2, []Vec{{120, 110.5}, {120, 108}}, nil, Vec{}},
	},
}

// shifted is the smallest graph of 32,300 whose content has an offset, a random
// one run through elkjs the same way.
var shifted = frame{
	size:   Vec{91, 136},
	offset: Vec{0, 3.9867067337036133},
	nodes: []frameNode{
		{"n1", 1, Vec{0, 0}, Vec{14, 14.013293266296387}, nil},
		{"n2", 2, Vec{0, 54.01329326629639}, Vec{14, 14.013293266296387}, []Vec{{14, 10}}},
		{"n0", 0, Vec{54, -3.9867067337036133}, Vec{37, 136}, []Vec{{0, 68}}},
	},
	edges: []frameEdge{{"e0", 0, 1, 0, 2, 0, nil, nil, Vec{}}},
}

func path(id string, points ...Vec) PlacedEdge {
	return PlacedEdge{ID: id, Points: points, Labels: []PlacedLabel{}}
}

func labelled(e PlacedEdge, l PlacedLabel) PlacedEdge {
	e.Labels = []PlacedLabel{l}
	return e
}

func TestFromLTRExport(t *testing.T) {
	tests := []struct {
		direction Direction
		frame     frame
		want      Result
	}{
		{Down, tall, Result{209.25, 224,
			[]PlacedNode{{"a", 97.25, 12, 100, 40}, {"b", 32, 182, 60.5, 30}, {"c", 117.25, 92, 80, 50}},
			[]PlacedEdge{
				labelled(path("e0", Vec{127.25, 52}, Vec{127.25, 72}, Vec{62.25, 72}, Vec{62.25, 182}), PlacedLabel{47.25, 112, 30, 10}),
				path("e1", Vec{147.25, 52}, Vec{147.25, 92}),
				path("e2", Vec{32, 196}, Vec{12, 196}, Vec{12, 198}, Vec{32, 198}),
				path("e3", Vec{167.25, 92}, Vec{167.25, 52}),
			}}},
		{Down, shifted, Result{160, 115,
			[]PlacedNode{{"n0", 12, 66, 136, 37}, {"n1", 15.986706733703613, 12, 14.013293266296387, 14}, {"n2", 70, 12, 14.013293266296387, 14}},
			[]PlacedEdge{path("e0", Vec{80, 26}, Vec{80, 66})}}},
		{Right, wide, Result{364.5, 154.5,
			[]PlacedNode{{"a", 12, 95, 100, 40}, {"b", 292, 32, 60.5, 30}, {"c", 172, 92.5, 80, 50}},
			[]PlacedEdge{
				labelled(path("e0", Vec{112, 110}, Vec{132, 110}, Vec{132, 47}, Vec{292, 47}), PlacedLabel{197, 41.5, 30, 10}),
				path("e1", Vec{112, 115}, Vec{152, 115}, Vec{152, 112.5}, Vec{172, 112.5}),
				path("e2", Vec{316.1666666666667, 32}, Vec{316.1666666666667, 12}, Vec{328.3333333333333, 12}, Vec{328.3333333333333, 32}),
				path("e3", Vec{172, 122.5}, Vec{132, 122.5}, Vec{132, 120}, Vec{112, 120}),
			}}},
		{Up, tall, Result{209.25, 224,
			[]PlacedNode{{"a", 97.25, 172, 100, 40}, {"b", 32, 12, 60.5, 30}, {"c", 117.25, 82, 80, 50}},
			[]PlacedEdge{
				labelled(path("e0", Vec{127.25, 172}, Vec{127.25, 152}, Vec{62.25, 152}, Vec{62.25, 42}), PlacedLabel{47.25, 102, 30, 10}),
				path("e1", Vec{147.25, 172}, Vec{147.25, 132}),
				path("e2", Vec{32, 28}, Vec{12, 28}, Vec{12, 26}, Vec{32, 26}),
				path("e3", Vec{167.25, 132}, Vec{167.25, 172}),
			}}},
		{Left, wide, Result{364.5, 154.5,
			[]PlacedNode{{"a", 252.5, 95, 100, 40}, {"b", 12, 32, 60.5, 30}, {"c", 112.5, 92.5, 80, 50}},
			[]PlacedEdge{
				labelled(path("e0", Vec{252.5, 110}, Vec{232.5, 110}, Vec{232.5, 47}, Vec{72.5, 47}), PlacedLabel{137.5, 41.5, 30, 10}),
				path("e1", Vec{252.5, 115}, Vec{212.5, 115}, Vec{212.5, 112.5}, Vec{192.5, 112.5}),
				path("e2", Vec{48.33333333333333, 32}, Vec{48.333333333333314, 12}, Vec{36.166666666666686, 12}, Vec{36.166666666666664, 32}),
				path("e3", Vec{192.5, 122.5}, Vec{232.5, 122.5}, Vec{232.5, 120}, Vec{252.5, 120}),
			}}},
	}
	for _, tt := range tests {
		g := tt.frame.graph(tt.direction)
		fromLTR(g)
		if got := export(g); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%d:\n got %+v\nwant %+v", tt.direction, got, tt.want)
		}
	}
}
