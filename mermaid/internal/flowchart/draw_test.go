package flowchart

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// Each case is a flowchart Chrome drew with Mermaid 12.3.0 (Arial, kerning and line
// hops off), and what it drew, in its viewBox's frame less the origin: where each
// line shows and its markers, each subgraph's rect, each edge label's middle, where
// each line of text starts, the title. Lengths are within 0.01; where text starts
// is within 1/16, as Blink sets it on a grid of 1/64.

type wantEdge struct {
	box    [4]float64 // the whole line's, as getBBox gives it, though it be invisible
	width  float64
	solids []float64 // each solid stretch's ends and the points 2 in from them: x, y, …
	dots   []float64 // how many dots, then the first's and the last's middles
	marks  []float64 // the points of the arrows' and crosses' outlines: x, y, …
	rings  []float64 // each circle's middle and radius: x, y, r, …
	cross  float64   // a cross's stroke width
}

type wantText struct {
	item   string    // "node A", "label L_A_B_0", "group s"
	top    float64   // the line's top: its baseline less the font's ascent
	starts []float64 // where each run of the line starts
}

type drawCase struct {
	name, src string
	size      [2]float64
	edges     map[string]wantEdge
	groups    map[string][4]float64
	labels    map[string][2]float64
	texts     []wantText
	title     []float64 // its middle and baseline
}

var drawCases = []drawCase{
	{
		name: "lines and markers", src: "flowchart LR\n  A[Start] -->|yes| B(Round)\n  B -.-> C{Choose}\n  C ==>|long| D([Stadium])\n  C --o E((Circle))\n  C --x F[(DB)]\n  A <--> G\n  G o--o H\n  H x--x I\n  I ~~~ J\n  J o-.-o K\n  K <-.-> L\n  L -.- M",
		size: [2]float64{1550.2269, 373.1372},
		edges: map[string]wantEdge{
			"L_A_B_0": {box: [4]float64{160, 118.2443, 229, 108.0298}, width: 2, solids: []float64{160, 226.2741, 162, 226.2741, 383, 118.2443, 385, 118.2443}, dots: []float64{},
				marks: []float64{378.5, 111.853, 389, 118.2443, 378.5, 124.6356}, rings: []float64{}, cross: 0},
			"L_B_C_0": {box: [4]float64{543, 118.2443, 37, 0}, width: 2, solids: []float64{575, 118.2443, 577, 118.2443, 577, 118.2443, 579, 118.2443}, dots: []float64{8, 544, 118.2443, 572, 118.2443},
				marks: []float64{569.5, 111.853, 580, 118.2443, 569.5, 124.6356}, rings: []float64{}, cross: 0},
			"L_C_D_0": {box: [4]float64{717.25, 30.5, 265.75, 49.9943}, width: 3.5, solids: []float64{717.25, 80.4943, 719.25, 80.4943, 977, 30.5, 979, 30.5}, dots: []float64{},
				marks: []float64{972.5, 24.1087, 983, 30.5, 972.5, 36.8913}, rings: []float64{}, cross: 0},
			"L_C_E_0": {box: [4]float64{755, 118.2443, 40, 0}, width: 2, solids: []float64{755, 118.2443, 757, 118.2443, 780.5, 118.2443, 782.5, 118.2443}, dots: []float64{},
				marks: []float64{}, rings: []float64{784.85, 118.2443, 7}, cross: 0},
			"L_C_F_0": {box: [4]float64{717.25, 155.9943, 77.75, 81.5686}, width: 2, solids: []float64{717.25, 155.9943, 719.25, 155.9943, 780.5042, 237.4648, 782.5, 237.5629}, dots: []float64{},
				marks: []float64{781.64, 232.3629, 792.04, 242.7629, 781.64, 242.7629, 792.04, 232.3629}, rings: []float64{}, cross: 2},
			"L_A_G_0": {box: [4]float64{164, 234.6074, 32, 108.0298}, width: 2, solids: []float64{168, 234.6074, 170, 234.6074, 190, 342.6372, 192, 342.6372}, dots: []float64{},
				marks: []float64{163, 234.6074, 174.5, 241.6074, 174.5, 227.6074, 185.5, 336.2459, 196, 342.6372, 185.5, 349.0285}, rings: []float64{}, cross: 0},
			"L_G_H_0": {box: [4]float64{352, 342.6372, 40, 0}, width: 2, solids: []float64{364.5, 342.6372, 366.5, 342.6372, 377.5, 342.6372, 379.5, 342.6372}, dots: []float64{},
				marks: []float64{}, rings: []float64{361.8, 342.6372, 7, 381.85, 342.6372, 7}, cross: 0},
			"L_H_I_0": {box: [4]float64{544, 342.6372, 49.5, 0}, width: 2, solids: []float64{556.5, 342.6372, 558.5, 342.6372, 579, 342.6372, 581, 342.6372}, dots: []float64{},
				marks: []float64{547.6, 337.4372, 558, 347.8372, 547.6, 347.8372, 558, 337.4372, 580.14, 337.4372, 590.54, 347.8372, 580.14, 347.8372, 590.54, 337.4372}, rings: []float64{}, cross: 2},
			"L_I_J_0": {box: [4]float64{745.5, 342.6372, 49.5, 0}, width: 0, solids: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_J_K_0": {box: [4]float64{947, 342.6372, 49.6135, 0}, width: 2, solids: []float64{983.5, 342.6372, 985.5, 342.6372, 994.6135, 342.6372, 996.6135, 342.6372}, dots: []float64{6, 960.5, 342.6372, 980.5, 342.6372},
				marks: []float64{}, rings: []float64{956.8, 342.6372, 7, 986.4635, 342.6372, 7}, cross: 0},
			"L_K_L_0": {box: [4]float64{1152.6134, 342.6372, 41.6135, 0}, width: 2, solids: []float64{1188.6134, 342.6372, 1190.6134, 342.6372, 1192.2269, 342.6372, 1194.2269, 342.6372}, dots: []float64{8, 1157.6134, 342.6372, 1185.6134, 342.6372},
				marks: []float64{1151.6134, 342.6372, 1163.1134, 349.6372, 1163.1134, 335.6372, 1183.7269, 336.2459, 1194.2269, 342.6372, 1183.7269, 349.0285}, rings: []float64{}, cross: 0},
			"L_L_M_0": {box: [4]float64{1350.2269, 342.6372, 40, 0}, width: 2, solids: []float64{}, dots: []float64{10, 1351.2269, 342.6372, 1387.2269, 342.6372},
				marks: []float64{}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{},
		labels: map[string][2]float64{
			"L_A_B_0": {276, 117.7443},
			"L_C_D_0": {871, 30},
		},
		texts: []wantText{
			{"node A", 221.9335, []float64{69.2222}},
			{"node B", 109.7407, []float64{447.366}},
			{"node C", 109.7407, []float64{645.3596}},
			{"node D", 21.9993, []float64{1046.9128}},
			{"node E", 109.7407, []float64{813.8483}},
			{"node F", 239.055, []float64{857.2589}},
			{"node G", 334.1263, []float64{270.5593}},
			{"node H", 334.1263, []float64{462.9436}},
			{"node I", 334.1263, []float64{667.5463}},
			{"node J", 334.1263, []float64{867.4929}},
			{"node K", 334.1263, []float64{1067.9279}},
			{"node L", 334.1263, []float64{1270.3003}},
			{"node M", 334.1263, []float64{1460.3566}},
			{"label L_A_B_0", 109.2407, []float64{265.1142}},
			{"label L_C_D_0", 21.4993, []float64{857.7589}},
		},
	},
	{
		name: "subgraphs and a title", src: "---\ntitle: Hello flowchart\n---\nflowchart TB\n  subgraph outer [Outer]\n    direction LR\n    subgraph inner [\"`**Bold** title that is long enough to wrap`\"]\n      x --> y\n    end\n  end\n  subgraph e [\" \"]\n  end\n  z>Odd] --> x",
		size: [2]float64{264, 474},
		edges: map[string]wantEdge{
			"L_x_y_0": {box: [4]float64{132, 349, 0, 20}, width: 2, solids: []float64{132, 349, 132, 351, 132, 363, 132, 365}, dots: []float64{},
				marks: []float64{138.3913, 358.5, 132, 369, 125.6087, 358.5}, rings: []float64{}, cross: 0},
			"L_z_x_0": {box: [4]float64{132, 106, 0, 194}, width: 2, solids: []float64{132, 106, 132, 108, 132, 294, 132, 296}, dots: []float64{},
				marks: []float64{138.3913, 289.5, 132, 300, 125.6087, 289.5}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{
			"e":     {1.375, 83.5, 8, 0},
			"outer": {8, 156, 248, 310},
			"inner": {28, 214, 208, 228},
		},
		labels: map[string][2]float64{},
		texts: []wantText{
			{"node x", 318, []float64{128.5}},
			{"node y", 387, []float64{128.5}},
			{"node z", 75, []float64{124.3906}},
			{"group outer", 158, []float64{114.4922}},
			{"group inner", 216, []float64{32.7656, 67.75}},
			{"group inner", 237, []float64{116.8125}},
		},
		title: []float64{132, 24},
	},
	{
		name: "a line shorter than its gaps", src: "flowchart LR\n  subgraph s\n    a\n  end\n  a o--o s",
		size: [2]float64{216, 133},
		edges: map[string]wantEdge{
			"L_a_s_0": {box: [4]float64{184, 78.5, 24, 0}, width: 2, solids: []float64{184, 78.5, 186, 78.5, 206, 78.5, 208, 78.5}, dots: []float64{},
				marks: []float64{}, rings: []float64{193.8, 78.5, 7, 197.85, 78.5, 7}, cross: 0},
		},
		groups: map[string][4]float64{
			"s": {8, 8, 200, 117},
		},
		labels: map[string][2]float64{},
		texts: []wantText{
			{"node a", 70, []float64{104.0938}},
			{"group s", 10, []float64{104.5}},
		},
	},
	{
		name: "a dotted line shorter than its gaps", src: "flowchart LR\n  subgraph s\n    a\n  end\n  a o-.-o s",
		size: [2]float64{216, 133},
		edges: map[string]wantEdge{
			"L_a_s_0": {box: [4]float64{184, 78.5, 24, 0}, width: 2, solids: []float64{196.5, 78.5, 198.5, 78.5, 206, 78.5, 208, 78.5}, dots: []float64{},
				marks: []float64{}, rings: []float64{193.8, 78.5, 7, 197.85, 78.5, 7}, cross: 0},
		},
		groups: map[string][4]float64{
			"s": {8, 8, 200, 117},
		},
		labels: map[string][2]float64{},
		texts: []wantText{
			{"node a", 70, []float64{104.0938}},
			{"group s", 10, []float64{104.5}},
		},
	},
	{
		name: "a gap into a corner", src: "flowchart TD\n  n1 --x n1\n  subgraph s1\n    n1\n  end",
		size: [2]float64{228, 133},
		edges: map[string]wantEdge{
			"L_n1_n1_0": {box: [4]float64{32, 75, 12, 7}, width: 2, solids: []float64{44, 75, 42, 75, 32.0076, 78.1814, 32.2615, 80.1518}, dots: []float64{},
				marks: []float64{30.64, 76.8, 41.04, 87.2, 30.64, 87.2, 41.04, 76.8}, rings: []float64{}, cross: 2},
		},
		groups: map[string][4]float64{
			"s1": {8, 8, 212, 117},
		},
		labels: map[string][2]float64{},
		texts: []wantText{
			{"node n1", 70, []float64{112.2031}},
			{"group s1", 10, []float64{106.6016}},
		},
	},
	{
		name: "dots round corners", src: "flowchart TB\n  A & B -.-> oC & D",
		size: [2]float64{360, 186},
		edges: map[string]wantEdge{
			"L_A_oC_0": {box: [4]float64{62, 53, 0, 76}, width: 2, solids: []float64{62, 125, 62, 127, 62, 127, 62, 129}, dots: []float64{18, 62, 54, 62, 122},
				marks: []float64{68.3913, 118.5, 62, 129, 55.6087, 118.5}, rings: []float64{}, cross: 0},
			"L_A_D_0": {box: [4]float64{106, 53, 148, 76}, width: 2, solids: []float64{254, 122.3283, 254, 124.3283, 254, 124.3283, 254, 126.3283, 254, 128.3283, 254, 129, 254, 128.3283, 254, 129}, dots: []float64{53, 106, 54, 254, 119.3283},
				marks: []float64{260.3913, 118.5, 254, 129, 247.6087, 118.5}, rings: []float64{}, cross: 0},
			"L_B_oC_0": {box: [4]float64{106, 53, 148, 76}, width: 2, solids: []float64{106, 123.6567, 106, 125.6567, 106, 125.6567, 106, 127.6567}, dots: []float64{52, 254, 54, 106, 120.6567},
				marks: []float64{112.3913, 118.5, 106, 129, 99.6087, 118.5}, rings: []float64{}, cross: 0},
			"L_B_D_0": {box: [4]float64{298, 53, 0, 76}, width: 2, solids: []float64{298, 125, 298, 127, 298, 127, 298, 129}, dots: []float64{18, 298, 54, 298, 122},
				marks: []float64{304.3913, 118.5, 298, 129, 291.6087, 118.5}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{},
		labels: map[string][2]float64{},
		texts: []wantText{
			{"node A", 22, []float64{79.3281}},
			{"node B", 22, []float64{271.3281}},
			{"node oC", 147, []float64{75.0469}},
			{"node D", 147, []float64{270.9375}},
		},
	},
	{
		name: "a point twice", src: "graph TD;A((chimpansen hoppar))-->C;",
		size: [2]float64{175.1377, 260.1377},
		edges: map[string]wantEdge{
			"L_A_C_0": {box: [4]float64{87.5689, 167.1377, 0, 36}, width: 2, solids: []float64{87.5689, 167.1377, 87.5689, 169.1377, 87.5689, 197.1377, 87.5689, 199.1377}, dots: []float64{},
				marks: []float64{93.9602, 192.6377, 87.5689, 203.1377, 81.1776, 192.6377}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{},
		labels: map[string][2]float64{},
		texts: []wantText{
			{"node A", 68.567, []float64{47.8623}},
			{"node A", 89.5654, []float64{65.7672}},
			{"node C", 221.1247, []float64{82.5004}},
		},
	},
	{
		name: "an invisible self loop", src: "flowchart TD\n  A ~~~ A",
		size: [2]float64{188, 61},
		edges: map[string]wantEdge{
			"L_A_A_0": {box: [4]float64{8, 27, 20, 7}, width: 0, solids: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{},
		labels: map[string][2]float64{},
		texts: []wantText{
			{"node A", 22, []float64{99.3281}},
		},
	},
	{
		name: "a subgraph of nothing", src: "flowchart LR\n  subgraph a [\" \"]\n  end",
		size:  [2]float64{16, 16},
		edges: map[string]wantEdge{},
		groups: map[string][4]float64{
			"a": {16, 20, 8, 0},
		},
		labels: map[string][2]float64{},
		texts:  []wantText{},
	},
	{
		name: "a point twice in a row", src: "flowchart TD\n  n0[\\\\marker/]\n  n1{{arrow an}}\n  n1 -. yes .-> n0\n  n0 o--o|marker| n1\n  n1 --- n2\n  n4 o--o n0\n  n2 --- n0\n  n2 ==> n4\n  n5 --- n2\n  n0 o--o|no| n1\n  subgraph s1 [\"`**Bold** group title that is long`\"]\n    n0\n    n1\n    subgraph s2\n      n2\n    end\n  end",
		size: [2]float64{372.3133, 839},
		edges: map[string]wantEdge{
			"L_n1_n0_0": {box: [4]float64{110.36, 484, 5.4, 137}, width: 2, solids: []float64{110.36, 488.3655, 110.36, 486.3655, 110.36, 486.3655, 110.36, 484.3655}, dots: []float64{34, 115.76, 620, 110.36, 491.3655},
				marks: []float64{103.9687, 494.5, 110.36, 484, 116.7513, 494.5}, rings: []float64{}, cross: 0},
			"L_n0_n1_0": {box: [4]float64{152.24, 480, 0, 141}, width: 2, solids: []float64{152.24, 492.5, 152.24, 494.5, 152.24, 606.5, 152.24, 608.5}, dots: []float64{},
				marks: []float64{}, rings: []float64{152.24, 489.8, 7, 152.24, 610.85, 7}, cross: 0},
			"L_n1_n2_0": {box: [4]float64{44.68, 202, 173.9, 419}, width: 2, solids: []float64{79.28, 621, 79.28, 619, 218.58, 288, 218.58, 290}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_n4_n0_0": {box: [4]float64{236, 480, 16.34, 306}, width: 2, solids: []float64{252.34, 773.5, 252.34, 771.5, 236, 494.5, 236, 492.5}, dots: []float64{},
				marks: []float64{}, rings: []float64{252.34, 776.2, 7, 236, 490.15, 7}, cross: 0},
			"L_n2_n0_0": {box: [4]float64{173.18, 335, 45.4, 94}, width: 2, solids: []float64{218.58, 335, 218.58, 337, 173.18, 427, 173.18, 429}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_n2_n4_0": {box: [4]float64{262.0467, 335, 34.2933, 447}, width: 3.5, solids: []float64{262.0467, 335, 262.0467, 337, 296.34, 776, 296.34, 778}, dots: []float64{},
				marks: []float64{302.7313, 771.5, 296.34, 782, 289.9487, 771.5}, rings: []float64{}, cross: 0},
			"L_n5_n2_0": {box: [4]float64{262.0467, 53, 0, 237}, width: 2, solids: []float64{262.0467, 53, 262.0467, 55, 262.0467, 288, 262.0467, 290}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_n0_n1_2": {box: [4]float64{188.72, 480, 5.4, 141}, width: 2, solids: []float64{194.12, 492.5, 194.12, 494.5, 188.72, 606.5, 188.72, 608.5}, dots: []float64{},
				marks: []float64{}, rings: []float64{194.12, 489.8, 7, 188.72, 610.85, 7}, cross: 0},
		},
		groups: map[string][4]float64{
			"s1": {8, 103, 356.3133, 633},
			"s2": {140.3133, 242, 200, 117},
		},
		labels: map[string][2]float64{
			"L_n1_n0_0": {115.76, 550.5},
			"L_n0_n1_0": {152.24, 550.5},
			"L_n0_n1_2": {194.12, 550.5},
		},
		texts: []wantText{
			{"node n0", 445.9999, []float64{149.4453}},
			{"node n1", 657.9995, []float64{106.7654}},
			{"node n2", 304.0002, []float64{232.5159}},
			{"node n4", 799.9991, []float64{266.5425}},
			{"node n5", 22.0009, []float64{254.2492}},
			{"label L_n1_n0_0", 541.9998, []float64{104.8613}},
			{"label L_n0_n1_0", 541.9998, []float64{130.4585}},
			{"label L_n0_n1_2", 541.9998, []float64{186.3305}},
			{"group s1", 105.0007, []float64{102.1096, 137.0939}},
			{"group s2", 244.0004, []float64{232.9144}},
		},
	},
}

func itemKey(it scene.Item) string {
	switch {
	case it.Kind == scene.Node:
		return "node " + it.ID
	case it.Kind == scene.Group:
		return "group " + it.ID
	case it.Kind == scene.Edge:
		return "edge " + it.ID
	case it.ID != "":
		return "label " + it.ID
	}
	return "title"
}

// edgeOf is what an edge's item draws, as wantEdge has it.
func edgeOf(it scene.Item) wantEdge {
	var g wantEdge
	var dots []scene.Point
	for _, p := range it.Paths {
		moves := 0
		curved := false
		for _, s := range p.Segs {
			if s.Op == scene.MoveTo {
				moves++
			}
			curved = curved || s.Op == scene.CubeTo
		}
		switch {
		case p.Fill != (scene.Paint{}) && curved:
			b := bounds(p.Segs)
			g.rings = append(g.rings, b.X+b.W/2, b.Y+b.H/2, b.W/2)
		case p.Fill != (scene.Paint{}) || moves > 1:
			if moves > 1 {
				g.cross = p.Stroke.Width
			}
			for _, s := range p.Segs {
				if s.Op == scene.MoveTo || s.Op == scene.LineTo {
					g.marks = append(g.marks, s.Pts[0].X, s.Pts[0].Y)
				}
			}
		case len(p.Stroke.Dash) > 0:
			g.width = p.Stroke.Width
			for l := 0.0; l <= arcLength(p.Segs)+1e-9; l += p.Stroke.Dash[0] + p.Stroke.Dash[1] {
				dots = append(dots, along(p.Segs, l))
			}
		default:
			g.width = p.Stroke.Width
			l := arcLength(p.Segs)
			for _, at := range []float64{0, min(2, l), max(l-2, 0), l} {
				q := along(p.Segs, at)
				g.solids = append(g.solids, q.X, q.Y)
			}
		}
	}
	if n := len(dots); n > 0 {
		g.dots = []float64{float64(n), dots[0].X, dots[0].Y, dots[n-1].X, dots[n-1].Y}
	}
	return g
}

// along is the point at length l along segs.
func along(segs []scene.Seg, l float64) scene.Point {
	if rest := span(segs, l, math.Inf(1)); len(rest) > 0 {
		return rest[0].Pts[0]
	}
	bs := bends(segs)
	return bs[len(bs)-1].to
}

func near(got, want []float64, tol float64) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if math.Abs(got[i]-want[i]) > tol {
			return false
		}
	}
	return true
}

// holds is whether got has a point near each of want's, and as many.
func holds(got, want []float64) bool {
	if len(got) != len(want) {
		return false
	}
	for i := 0; i < len(want); i += 2 {
		found := false
		for j := 0; j < len(got); j += 2 {
			found = found || near(got[j:j+2], want[i:i+2], 0.01)
		}
		if !found {
			return false
		}
	}
	return true
}

// rows are an item's lines of text by their tops, each with where its runs start.
func rows(it scene.Item) map[float64][]float64 {
	out := map[float64][]float64{}
	for _, t := range it.Texts {
		a, _ := extent(scene.Arial, t.Font)
		for _, l := range t.Lines {
			out[l.Baseline-a] = append(out[l.Baseline-a], l.X)
		}
	}
	for top := range out {
		slices.Sort(out[top])
	}
	return out
}

// rank is where paintLayoutData stacks an item: subgraphs, edges, edge labels,
// nodes, then the title.
func rank(it scene.Item) int {
	return map[string]int{"group": 0, "edge": 1, "label": 2, "node": 3, "title": 4}[strings.Fields(itemKey(it))[0]]
}

func TestDraw(t *testing.T) {
	for _, c := range drawCases {
		body, title := syntax.Prepare(c.src)
		s, err := Layout(body, title, scene.Arial)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !near([]float64{s.Width, s.Height}, c.size[:], 0.01) {
			t.Errorf("%s: scene %v x %v, want %v", c.name, s.Width, s.Height, c.size)
		}
		items := map[string]scene.Item{}
		for i, it := range s.Items {
			items[itemKey(it)] = it
			if i > 0 && rank(s.Items[i-1]) > rank(it) {
				t.Errorf("%s: %s drawn after %s", c.name, itemKey(s.Items[i-1]), itemKey(it))
			}
		}
		for id, w := range c.edges {
			it := items["edge "+id]
			g := edgeOf(it)
			g.box = [4]float64{it.Box.X, it.Box.Y, it.Box.W, it.Box.H}
			if !near(g.box[:], w.box[:], 0.01) || math.Abs(g.width-w.width) > 0.01 || !near(g.solids, w.solids, 0.01) || !near(g.dots, w.dots, 0.01) ||
				!holds(g.marks, w.marks) || !near(g.rings, w.rings, 0.01) || math.Abs(g.cross-w.cross) > 0.01 {
				t.Errorf("%s: edge %s draws\n %+v\nwant %+v", c.name, id, g, w)
			}
		}
		for id, w := range c.groups {
			if b := bounds(items["group "+id].Paths[0].Segs); !near([]float64{b.X, b.Y, b.W, b.H}, w[:], 0.01) {
				t.Errorf("%s: subgraph %s at %v, want %v", c.name, id, b, w)
			}
		}
		for id, w := range c.labels {
			if b := items["label "+id].Box; !near([]float64{b.X + b.W/2, b.Y + b.H/2}, w[:], 0.01) {
				t.Errorf("%s: edge label %s about %v, want %v", c.name, id, b, w)
			}
		}
		for _, w := range c.texts {
			var got []float64
			for top, starts := range rows(items[w.item]) {
				if math.Abs(top-w.top) <= 1.0/32 {
					got = starts
				}
			}
			if !near(got, w.starts, 1.0/16) {
				t.Errorf("%s: %s's line at %v starts at %v, want %v", c.name, w.item, w.top, got, w.starts)
			}
		}
		if c.title != nil {
			it := items["title"]
			if got := []float64{it.Box.X + it.Box.W/2, it.Texts[0].Lines[0].Baseline}; !near(got, c.title, 0.01) {
				t.Errorf("%s: title at %v, want %v", c.name, got, c.title)
			}
		}
	}
}
