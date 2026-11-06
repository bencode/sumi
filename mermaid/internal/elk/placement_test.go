// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The positions expected below come from elkjs 0.12.1, on nodes 30 wide and as
// high as placedNodes makes them.

// placedNodes runs spec up to placeNodes, with nodes of several heights, a low
// one among them for margins.
func placedNodes(spec string) *Graph {
	g := layered(spec)
	heights := []float64{45, 30, 5, 62, 14}
	for _, n := range nodes(g) {
		if n.Type == typeNormal {
			n.Size = Vec{30, heights[int(n.ID[0])%len(heights)]}
		}
	}
	sortByModelOrder(g)
	minimizeCrossings(g)
	positionPorts(g)
	nodeMargins(g)
	placeNodes(g)
	return g
}

// positions writes the nodes of g like "a=0,.=65 | b=12", layer by layer, a
// long edge's nodes as dots.
func positions(g *Graph) string {
	layers := make([]string, len(g.Layers))
	for i, l := range g.Layers {
		all := make([]string, len(l.Nodes))
		for j, n := range l.Nodes {
			name := n.ID
			if n.Type == typeLongEdge {
				name = "."
			}
			all[j] = name + "=" + strconv.FormatFloat(n.Pos.Y, 'f', -1, 64)
		}
		layers[i] = strings.Join(all, ",")
	}
	return strings.Join(layers, " | ")
}

// Each rule of the placement changes the outcome of one of these at least.
func TestPlacement(t *testing.T) {
	tests := []struct{ spec, want string }{
		{
			"a>b b>c b>d d>e c>d",
			"a=21 | b=0 | c=9,.=43 | d=8.499999999999996 | e=15.999999999999996",
		},
		{
			"a>b a>c b>d c>b a>b c>d",
			"a=26.083333333333336 | .=2.3333333333333357,.=23.333333333333336,c=68.33333333333334 | b=-7.666666666666664,.=74.33333333333334 | d=26.333333333333332",
		},
		{
			"a>b b>c c>d c>e a>f b>d a>f c>d",
			"a=-41.375 | f=-45,b=0 | c=1.75,.=38 | e=-60.75,d=9.25",
		},
		{
			"a>b a>c b>d c>e a>f c>f b>f b>c",
			"a=-85.125 | .=-138,b=-110.25,.=3.375 | c=-146,.=-82.625,d=-61.625,.=3.375 | e=-153,f=-81.75",
		},
		{
			"a>b a>c b>d b>e a>f b>g d>g c>f",
			"a=168 | b=45.875,c=150.5,.=184.5 | d=-10.5,e=74.5,.=136.5,f=179.375 | g=24.375",
		},
		{
			"a>b a>c b>d d>e c>f c>e a>d e>f c>e",
			"a=78.91666666666669 | .=6.500000000000007,b=32.66666666666667,.=114.66666666666669 | c=-3.499999999999993,d=79.33333333333334 | .=6.75,e=42.5 | f=3.000000000000007",
		},
		{
			"a>b a>c b>c b>c a>c d>e d>f f>g f>h f>h",
			"a=52.125,d=143.91666666666669 | b=-29.174999999999983,.=52.82500000000002,.=94.75,e=115.75,f=187 | c=16.37500000000003,g=81.16666666666669,h=183.16666666666669",
		},
		{
			"a>b c>b a>d d>e a>f e>g g>h d>i c>i e>i c>d",
			"a=58.20833333333334,c=158.375 | f=-3.75,d=46.25,b=131.25,.=213.25 | e=32.375000000000014,.=114.66666666666669,.=213.25 | g=14.70833333333335,i=137.66666666666669 | h=35.70833333333335",
		},
		{
			"a>b a>c b>d d>e e>f b>g b>g c>f c>g b>g a>g a>e",
			"a=132.53333333333333 | b=8.88333333333333,.=90.88333333333333,c=111.88333333333333,.=158.83333333333331 | d=-8.166666666666671,g=76.83333333333333,.=158.83333333333331,.=186 | e=71.58333333333333,.=186 | f=101.58333333333333",
		},
		{
			"a>b a>c d>e d>f e>g e>h e>i e>j d>k f>i j>g f>e",
			"a=-178.62500000000003,d=-70.16666666666667 | b=-243.66666666666669,c=-141.66666666666669,k=-87.66666666666669,.=-57.66666666666668,f=53.66666666666668 | e=-71.00000000000001,.=53.66666666666668 | h=-98,.=-64,j=-43,i=27 | g=-77",
		},
		{
			"a>b c>a a>d b>e b>f g>e g>h e>h c>f a>b b>f b>d a>b",
			"c=-47.79166666666667 | .=-91.55000000000001,a=-15.041666666666668 | g=-173.55,.=-91.55000000000001,b=-70.55000000000001,.=40.141666666666666 | .=-159.7166666666667,e=-138.7166666666667,f=-45.416666666666664,d=13.474999999999987 | h=-153.95833333333331",
		},
		{
			"b>a b>c b>d e>b f>b g>c b>h e>i b>j g>k b>f b>d e>i",
			"e=-61.249999999999986 | g=-363,b=-172.16666666666666,i=-70.16666666666666 | k=-356,c=-306,a=-252,d=-202,h=-117,j=-62.99999999999999,f=7.000000000000007",
		},
		{
			"a>b c>a d>a c>e e>f e>g d>e a>f b>c a>c a>f e>f a>c",
			"d=65.54761904761904 | a=46.933333333333344,.=180.58333333333331 | .=-12.345238095238102,.=8.654761904761898,b=29.654761904761905,.=111.6547619047619,.=132.6547619047619,.=153.6547619047619,.=180.58333333333331 | .=-12.345238095238102,.=8.654761904761898,c=89.0547619047619,.=180.58333333333331 | .=-12.345238095238102,.=8.654761904761898,e=101.12857142857143 | f=18.595238095238095,g=126.37857142857143",
		},
		{
			"b>a c>a c>d e>c e>f g>a h>d i>f j>f h>k g>l g>m h>n a>e",
			"b=-151.5,g=-49.5 | a=-125.5,l=-60,m=42 | .=-136,e=-115,i=-45,j=40 | c=-128.5,f=-56.875,h=-3.25 | d=-136.83333333333334,k=-31.5,n=18.5",
		},
		{
			"a>b c>b b>d d>e f>b g>f a>h e>i d>j k>d h>l f>k a>k e>h b>i",
			"g=59.5 | c=5.083333333333329,f=80.5,a=142.875 | b=3.1666666666666714,k=123,.=191.66666666666666 | .=27.16666666666667,d=106.33333333333331,.=191.66666666666666 | .=27.16666666666667,e=70,j=140,.=191.66666666666666 | i=32.499999999999986,h=147 | l=126",
		},
		{
			"a>b b>c c>d d>e b>f c>g h>b i>c c>j k>h c>i b>d a>f b>g g>h a>e f>k",
			"a=-60.83333333333334 | .=-357.33333333333337,b=-89.33333333333334,.=12.791666666666657 | .=-357.33333333333337,.=-318,c=-229.66666666666669,.=-58.33333333333334,.=-14.333333333333343,f=12.791666666666657 | .=-357.33333333333337,d=-336.33333333333337,i=-251.33333333333334,j=-166.33333333333334,g=-96.33333333333334,.=-14.333333333333343,k=12.791666666666657 | e=-355.95833333333337,h=-26.708333333333343",
		},
		{
			"a>b b>c b>d b>e c>f c>g d>f a>c a>b d>f h>i i>j h>k j>l k>m j>n i>o k>p p>q p>r i>o i>k h>j j>m",
			"a=-358.5,h=-111.83333333333333 | b=-395.75,.=-302,i=-194.83333333333337,.=-104.83333333333333,.=1 | d=-463,e=-378,c=-308,o=-191.50000000000003,j=-121.5,k=1 | f=-444.875,g=-333,l=-231,n=-129,m=-44,p=10 | q=-59,r=43",
		},
		{
			"a>b a>c c>d a>e c>f b>g a>h e>f a>d b>e i>j j>k k>l k>m k>n j>o j>p j>n j>l k>m i>l j>o q>r r>s s>t t>u s>v u>w u>x q>y q>z z>n26 q>n26 u>z v>n26",
			"i=76.73214285714282,q=361.91666666666663,a=640.8333333333333 | j=64.74999999999997,.=213.1666666666666,.=243.9166666666666,r=352.16666666666663,y=406.16666666666663,.=478.33333333333326,.=517.6666666666665,c=558.3333333333333,b=612.3333333333333,.=723.9999999999999,h=744.9999999999999 | .=9.416666666666607,o=30.416666666666607,k=102.4166666666666,.=162.1666666666666,p=183.1666666666666,.=213.1666666666666,.=243.9166666666666,s=339.66666666666663,.=478.33333333333326,d=499.33333333333326,.=564.3333333333333,g=605.3333333333333,e=707.3333333333333 | n=-8.916666666666746,m=97.4166666666666,l=151.4166666666666,.=243.9166666666666,t=311.16666666666663,.=407.91666666666663,.=478.33333333333326,f=639.5833333333333 | .=243.9166666666666,u=316.16666666666663,v=376.91666666666663,.=478.33333333333326 | z=238.9166666666666,x=288.91666666666663,w=373.91666666666663,.=407.91666666666663,.=478.33333333333326 | n26=384.16666666666663",
		},
		{
			"a>b b>c b>d b>e b>f a>g d>h d>i g>j b>j a>i a>g b>g b>d e>g",
			"a=225.66666666666669 | .=11.33333333333331,b=132,.=300.73333333333335,.=321.73333333333335 | .=11.33333333333331,c=32.333333333333314,f=86.33333333333331,d=136.33333333333331,.=201.33333333333331,e=224.03333333333333,.=274.03333333333336,.=300.73333333333335,.=321.73333333333335 | i=-7.000000000000014,h=153,.=201.33333333333331,g=260.1583333333333 | j=238.55833333333334",
		},
	}
	for _, tt := range tests {
		if got := positions(placedNodes(tt.spec)); got != tt.want {
			t.Errorf("%s:\n got %s\nwant %s", tt.spec, got, tt.want)
		}
	}
}

// The balance was sound in every graph compared with elkjs, so the rest of the
// choice shows only here. The layouts are made up, and what is expected follows
// p4nodes/bk/BKNodePlacer.java lines 239 to 267.
func TestChoice(t *testing.T) {
	a := &Node{Size: Vec{Y: 10}, Margin: Spacing{Top: 2, Bottom: 2}}
	b := &Node{Size: Vec{Y: 10}, Margin: Spacing{Top: 2, Bottom: 2}}
	g := &Graph{Settings: rootSettings(), Layers: []*Layer{{Nodes: []*Node{a, b}}}}
	layouts := func(ys [4][2]float64) []*alignment {
		all := make([]*alignment, len(ys))
		for i, y := range ys {
			all[i] = newAlignment(false, false)
			for j, n := range []*Node{a, b} {
				all[i].y[n], all[i].root[n], all[i].size[n] = y[j], n, n.Size.Y
			}
		}
		return all
	}
	tests := []struct {
		name string
		ys   [4][2]float64
		want int // the layout chosen, or -1 for their balance
	}{
		{"balance", [4][2]float64{{0, 100}, {0, 20}, {0, 30}, {0, 40}}, -1},
		{"smallest sound", [4][2]float64{{0, 100}, {0, 20}, {12, 0}, {12, 0}}, 1},
		{"first of equals", [4][2]float64{{0, 20}, {0, 20}, {12, 0}, {12, 0}}, 0},
		{"none sound", [4][2]float64{{12, 0}, {12, 0}, {0, 14}, {0, 14}}, 0},
		{"touching margins", [4][2]float64{{0, 14}, {0, 14}, {0, 14}, {0, 14}}, 0},
		{"margins apart", [4][2]float64{{0, 15}, {0, 15}, {0, 15}, {0, 15}}, -1},
	}
	for _, tt := range tests {
		all := layouts(tt.ys)
		got := choose(g, all)
		if tt.want >= 0 && got != all[tt.want] {
			t.Errorf("%s: chose %v, want layout %d", tt.name, got.y, tt.want)
		}
		if tt.want < 0 && slices.Contains(all, got) {
			t.Errorf("%s: chose a layout, want their balance", tt.name)
		}
	}
}

func TestNodeSpacing(t *testing.T) {
	tests := []struct {
		a, b NodeType
		want float64
	}{
		{typeNormal, typeNormal, 40},
		{typeNormal, typeLabel, 40},
		{typeLabel, typeNormal, 40},
		{typeNormal, typeLongEdge, 20},
		{typeLongEdge, typeNormal, 20},
		{typeLongEdge, typeLongEdge, 20},
		{typeLongEdge, typeLabel, 20},
		{typeLabel, typeLongEdge, 20},
		{typeLabel, typeLabel, 20},
	}
	for _, tt := range tests {
		if got := nodeSpacing(rootSettings(), &Node{Type: tt.a}, &Node{Type: tt.b}); got != tt.want {
			t.Errorf("%d %d: got %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
