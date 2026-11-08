package flowchart

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/line"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// Each case is a flowchart Chrome drew with Mermaid 12.3.0 (Arial, kerning
// off), and what it drew, in its viewBox's frame less the origin: where each
// line shows, its hops and its markers, each subgraph's rect, each edge label's
// middle, where each line of text starts, the title. Lengths are within 0.01, a hop
// within 0.1, as Blink finds its center in float32, and where text starts within
// 1/16, as Blink sets it on a grid of 1/64.

type wantEdge struct {
	box    [4]float64 // the whole line's, as getBBox gives it, though it be invisible
	width  float64
	solids []float64 // each solid stretch's ends and the points 2 in from them: x, y, …
	hops   []float64 // each hop's start, top, end and its first curve's middle: x, y, …
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

// wantStyle is what a diagram's styles paint on an item. A color is "#rrggbbaa", ""
// for the theme's; a text's may be "on", the black or white that reads on the item's
// fill, sumi's choice where Chrome keeps the theme's.
type wantStyle struct {
	fill, stroke, text string
	width              float64   // of the stroke, none when it paints nothing
	marks              []string  // a line's markers, its start's first
	dashes             []float64 // how many, then the first's ends and the last's: x, y, …
}

type drawCase struct {
	name, src string
	size      [2]float64
	edges     map[string]wantEdge
	groups    map[string][4]float64
	labels    map[string][2]float64
	texts     []wantText
	title     []float64 // its middle and baseline
	styles    map[string]wantStyle
}

var drawCases = []drawCase{
	{
		name: "lines and markers", src: "flowchart LR\n  A[Start] -->|yes| B(Round)\n  B -.-> C{Choose}\n  C ==>|long| D([Stadium])\n  C --o E((Circle))\n  C --x F[(DB)]\n  A <--> G\n  G o--o H\n  H x--x I\n  I ~~~ J\n  J o-.-o K\n  K <-.-> L\n  L -.- M",
		size: [2]float64{1550.2269, 373.1372},
		edges: map[string]wantEdge{
			"L_A_B_0": {box: [4]float64{160, 118.2443, 229, 108.0298}, width: 2, solids: []float64{160, 226.2741, 162, 226.2741, 382.9995, 118.2443, 384.9995, 118.2443}, hops: []float64{}, dots: []float64{},
				marks: []float64{378.5, 111.853, 389, 118.2443, 378.5, 124.6356}, rings: []float64{}, cross: 0},
			"L_B_C_0": {box: [4]float64{543, 118.2443, 37, 0}, width: 2, solids: []float64{575, 118.2443, 577, 118.2443, 577, 118.2443, 579, 118.2443}, hops: []float64{}, dots: []float64{8, 544, 118.2443, 572, 118.2443},
				marks: []float64{569.5, 111.853, 580, 118.2443, 569.5, 124.6356}, rings: []float64{}, cross: 0},
			"L_C_D_0": {box: [4]float64{717.25, 30.5, 265.75, 49.9943}, width: 3.5, solids: []float64{717.25, 80.4943, 719.25, 80.4943, 977, 30.5, 979, 30.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{972.5, 24.1087, 983, 30.5, 972.5, 36.8913}, rings: []float64{}, cross: 0},
			"L_C_E_0": {box: [4]float64{755, 118.2443, 40, 0}, width: 2, solids: []float64{755, 118.2443, 757, 118.2443, 780.5, 118.2443, 782.5, 118.2443}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{784.85, 118.2443, 7}, cross: 0},
			"L_C_F_0": {box: [4]float64{717.25, 155.9943, 77.75, 81.5686}, width: 2, solids: []float64{717.25, 155.9943, 719.25, 155.9943, 780.5039, 237.4648, 782.4996, 237.5629}, hops: []float64{}, dots: []float64{},
				marks: []float64{781.64, 232.3629, 792.04, 242.7629, 781.64, 242.7629, 792.04, 232.3629}, rings: []float64{}, cross: 2},
			"L_A_G_0": {box: [4]float64{164, 234.6074, 32, 108.0298}, width: 2, solids: []float64{168, 234.6074, 170, 234.6074, 190.0004, 342.6372, 192.0004, 342.6372}, hops: []float64{}, dots: []float64{},
				marks: []float64{163, 234.6074, 174.5, 241.6074, 174.5, 227.6074, 185.5, 336.2459, 196, 342.6372, 185.5, 349.0285}, rings: []float64{}, cross: 0},
			"L_G_H_0": {box: [4]float64{352, 342.6372, 40, 0}, width: 2, solids: []float64{364.5, 342.6372, 366.5, 342.6372, 377.5, 342.6372, 379.5, 342.6372}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{361.8, 342.6372, 7, 381.85, 342.6372, 7}, cross: 0},
			"L_H_I_0": {box: [4]float64{544, 342.6372, 49.5, 0}, width: 2, solids: []float64{556.5, 342.6372, 558.5, 342.6372, 579, 342.6372, 581, 342.6372}, hops: []float64{}, dots: []float64{},
				marks: []float64{547.6, 337.4372, 558, 347.8372, 547.6, 347.8372, 558, 337.4372, 580.14, 337.4372, 590.54, 347.8372, 580.14, 347.8372, 590.54, 337.4372}, rings: []float64{}, cross: 2},
			"L_I_J_0": {box: [4]float64{745.5, 342.6372, 49.5, 0}, width: 0, solids: []float64{}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_J_K_0": {box: [4]float64{947, 342.6372, 49.6135, 0}, width: 2, solids: []float64{983.5, 342.6372, 985.5, 342.6372, 994.6135, 342.6372, 996.6135, 342.6372}, hops: []float64{}, dots: []float64{6, 960.5, 342.6372, 980.5, 342.6372},
				marks: []float64{}, rings: []float64{956.8, 342.6372, 7, 986.4635, 342.6372, 7}, cross: 0},
			"L_K_L_0": {box: [4]float64{1152.6134, 342.6372, 41.6135, 0}, width: 2, solids: []float64{1188.6134, 342.6372, 1190.6134, 342.6372, 1192.2269, 342.6372, 1194.2269, 342.6372}, hops: []float64{}, dots: []float64{8, 1157.6134, 342.6372, 1185.6134, 342.6372},
				marks: []float64{1151.6134, 342.6372, 1163.1134, 349.6372, 1163.1134, 335.6372, 1183.7269, 336.2459, 1194.2269, 342.6372, 1183.7269, 349.0285}, rings: []float64{}, cross: 0},
			"L_L_M_0": {box: [4]float64{1350.2269, 342.6372, 40, 0}, width: 2, solids: []float64{}, hops: []float64{}, dots: []float64{10, 1351.2269, 342.6372, 1387.2269, 342.6372},
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
			"L_x_y_0": {box: [4]float64{132, 349, 0, 20}, width: 2, solids: []float64{132, 349, 132, 351, 132, 363, 132, 365}, hops: []float64{}, dots: []float64{},
				marks: []float64{138.3913, 358.5, 132, 369, 125.6087, 358.5}, rings: []float64{}, cross: 0},
			"L_z_x_0": {box: [4]float64{132, 106, 0, 194}, width: 2, solids: []float64{132, 106, 132, 108, 132, 294, 132, 296}, hops: []float64{}, dots: []float64{},
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
			"L_a_s_0": {box: [4]float64{184, 78.5, 24, 0}, width: 2, solids: []float64{184, 78.5, 186, 78.5, 206, 78.5, 208, 78.5}, hops: []float64{}, dots: []float64{},
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
			"L_a_s_0": {box: [4]float64{184, 78.5, 24, 0}, width: 2, solids: []float64{196.5, 78.5, 198.5, 78.5, 206, 78.5, 208, 78.5}, hops: []float64{}, dots: []float64{},
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
			"L_n1_n1_0": {box: [4]float64{32, 75, 12, 7}, width: 2, solids: []float64{44, 75, 42, 75, 32.0076, 78.1814, 32.2615, 80.1518}, hops: []float64{}, dots: []float64{},
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
			"L_A_oC_0": {box: [4]float64{62, 53, 0, 76}, width: 2, solids: []float64{62, 125, 62, 127, 62, 127, 62, 129}, hops: []float64{}, dots: []float64{18, 62, 54, 62, 122},
				marks: []float64{68.3913, 118.5, 62, 129, 55.6087, 118.5}, rings: []float64{}, cross: 0},
			"L_A_D_0": {box: [4]float64{106, 53, 148, 76}, width: 2, solids: []float64{106, 53, 106, 55, 254, 125, 254, 127}, hops: []float64{174, 93, 180, 87, 186, 93, 175.7574, 88.7574}, dots: []float64{},
				marks: []float64{260.3913, 118.5, 254, 129, 247.6087, 118.5}, rings: []float64{}, cross: 0},
			"L_B_oC_0": {box: [4]float64{106, 53, 148, 76}, width: 2, solids: []float64{106, 123.6567, 106, 125.6567, 106, 125.6567, 106, 127.6567}, hops: []float64{}, dots: []float64{52, 254, 54, 106, 120.6567},
				marks: []float64{112.3913, 118.5, 106, 129, 99.6087, 118.5}, rings: []float64{}, cross: 0},
			"L_B_D_0": {box: [4]float64{298, 53, 0, 76}, width: 2, solids: []float64{298, 125, 298, 127, 298, 127, 298, 129}, hops: []float64{}, dots: []float64{18, 298, 54, 298, 122},
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
			"L_A_C_0": {box: [4]float64{87.5689, 167.1377, 0, 36}, width: 2, solids: []float64{87.5689, 167.1377, 87.5689, 169.1377, 87.5689, 197.1377, 87.5689, 199.1377}, hops: []float64{}, dots: []float64{},
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
			"L_A_A_0": {box: [4]float64{8, 27, 20, 7}, width: 0, solids: []float64{}, hops: []float64{}, dots: []float64{},
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
		name: "node styles", src: "flowchart LR\n  A:::c --> B --> C --> D --> E --> F\n  classDef c fill:#f00\n  style A fill:#0f0,fill:#00f\n  style B Fill:#fde68a,STROKE:#b45309\n  style C fill:#1e3a8a,color:#fca5a5\n  style D fill:#33669980,stroke:none,stroke-width:3px\n  style E color:none,fill:#000\n  style F fill:#abc,stroke:#0a0f,stroke-width:1px",
		size: [2]float64{1128, 61},
		edges: map[string]wantEdge{
			"L_A_B_0": {box: [4]float64{160, 30.5, 36, 0}, width: 2, solids: []float64{160, 30.5, 162, 30.5, 190, 30.5, 192, 30.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{185.5, 24.1087, 196, 30.5, 185.5, 36.8913}, rings: []float64{}, cross: 0},
			"L_B_C_0": {box: [4]float64{352, 30.5, 36, 0}, width: 2, solids: []float64{352, 30.5, 354, 30.5, 382, 30.5, 384, 30.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{377.5, 24.1087, 388, 30.5, 377.5, 36.8913}, rings: []float64{}, cross: 0},
			"L_C_D_0": {box: [4]float64{544, 30.5, 36, 0}, width: 2, solids: []float64{544, 30.5, 546, 30.5, 574, 30.5, 576, 30.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{569.5, 24.1087, 580, 30.5, 569.5, 36.8913}, rings: []float64{}, cross: 0},
			"L_D_E_0": {box: [4]float64{736, 30.5, 36, 0}, width: 2, solids: []float64{736, 30.5, 738, 30.5, 766, 30.5, 768, 30.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{761.5, 24.1087, 772, 30.5, 761.5, 36.8913}, rings: []float64{}, cross: 0},
			"L_E_F_0": {box: [4]float64{928, 30.5, 36, 0}, width: 2, solids: []float64{928, 30.5, 930, 30.5, 958, 30.5, 960, 30.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{953.5, 24.1087, 964, 30.5, 953.5, 36.8913}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{},
		labels: map[string][2]float64{},
		texts: []wantText{
			{"node A", 22, []float64{79.3281}},
			{"node B", 22, []float64{271.3281}},
			{"node C", 22, []float64{462.9375}},
			{"node D", 22, []float64{654.9375}},
			{"node E", 22, []float64{847.3281}},
			{"node F", 22, []float64{1039.7188}},
		},
		styles: map[string]wantStyle{
			"node A":       {fill: "#0000ffff", stroke: "", text: "on", width: 2},
			"node B":       {fill: "#fde68aff", stroke: "#b45309ff", text: "on", width: 2},
			"node C":       {fill: "#1e3a8aff", stroke: "", text: "#fca5a5ff", width: 2},
			"node D":       {fill: "#33669980", stroke: "#00000000", text: "", width: 0},
			"node E":       {fill: "#000000ff", stroke: "", text: "on", width: 2},
			"node F":       {fill: "#aabbccff", stroke: "#00aa00ff", text: "on", width: 1},
			"edge L_A_B_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
			"edge L_B_C_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
			"edge L_C_D_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
			"edge L_D_E_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
			"edge L_E_F_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
		},
	},
	{
		name: "subgraph styles", src: "flowchart LR\n  subgraph s [Title]\n    a --> b\n  end\n  style s fill:#ecfeff,stroke:#0891b2,stroke-dasharray:6 3,color:#0e7490\n  subgraph t [Dark]\n    c\n  end\n  style t fill:#111,stroke:#444",
		size: [2]float64{392, 290},
		edges: map[string]wantEdge{
			"L_a_b_0": {box: [4]float64{184, 235.5, 20, 0}, width: 2, solids: []float64{184, 235.5, 186, 235.5, 198, 235.5, 200, 235.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{193.5, 229.1087, 204, 235.5, 193.5, 241.8913}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{
			"t": {96, 8, 200, 117},
			"s": {8, 165, 376, 117},
		},
		labels: map[string][2]float64{},
		texts: []wantText{
			{"node a", 227, []float64{104.0938}},
			{"node b", 227, []float64{280.0938}},
			{"node c", 70, []float64{192.5}},
			{"group t", 10, []float64{181.2188}},
			{"group s", 167, []float64{182.7734}},
		},
		styles: map[string]wantStyle{
			"node a":       {fill: "", stroke: "", text: "", width: 2},
			"node b":       {fill: "", stroke: "", text: "", width: 2},
			"node c":       {fill: "", stroke: "", text: "", width: 2},
			"group t":      {fill: "#111111ff", stroke: "#444444ff", text: "on", width: 1},
			"group s":      {fill: "#ecfeffff", stroke: "#0891b2ff", text: "#0e7490ff", width: 1, dashes: []float64{110, 8.5, 165, 13.5, 165, 8, 169.5, 8, 165.5}},
			"edge L_a_b_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
		},
	},
	{
		name: "dashes on shapes", src: "flowchart LR\n  A[r] --> B((c)) --> C([s]) --> D>f] --> E{{h}} --> F[z]\n  style A stroke-dasharray:5 5\n  style B stroke-dasharray:6 2,stroke-width:3\n  style C stroke-dasharray:4 4\n  style D stroke-dasharray:6 2\n  style E stroke-dasharray:3,stroke-width:1\n  style F stroke-dasharray:0 4 6 4",
		size: [2]float64{1122.6128, 107},
		edges: map[string]wantEdge{
			"L_A_B_0": {box: [4]float64{160, 53.5, 36, 0}, width: 2, solids: []float64{160, 53.5, 162, 53.5, 190, 53.5, 192, 53.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{185.5, 47.1087, 196, 53.5, 185.5, 59.8913}, rings: []float64{}, cross: 0},
			"L_B_C_0": {box: [4]float64{254.136, 53.5, 36, 0}, width: 2, solids: []float64{254.136, 53.5, 256.136, 53.5, 284.136, 53.5, 286.136, 53.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{279.636, 47.1087, 290.136, 53.5, 279.636, 59.8913}, rings: []float64{}, cross: 0},
			"L_C_D_0": {box: [4]float64{465.3627, 53.5, 47.2501, 0}, width: 2, solids: []float64{465.3627, 53.5, 467.3627, 53.5, 506.6128, 53.5, 508.6129, 53.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{502.1129, 47.1087, 512.6129, 53.5, 502.1129, 59.8913}, rings: []float64{}, cross: 0},
			"L_D_E_0": {box: [4]float64{678.6129, 53.5, 36, 0}, width: 2, solids: []float64{678.6129, 53.5, 680.6129, 53.5, 708.6129, 53.5, 710.6129, 53.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{704.1129, 47.1087, 714.6129, 53.5, 704.1129, 59.8913}, rings: []float64{}, cross: 0},
			"L_E_F_0": {box: [4]float64{922.6129, 53.5, 36, 0}, width: 2, solids: []float64{922.6129, 53.5, 924.6129, 53.5, 952.6129, 53.5, 954.6129, 53.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{948.1129, 47.1087, 958.6129, 53.5, 948.1129, 59.8913}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{},
		labels: map[string][2]float64{},
		texts: []wantText{
			{"node A", 45, []float64{81.656}},
			{"node B", 45, []float64{223.5673}},
			{"node C", 45, []float64{376.2482}},
			{"node D", 45, []float64{595.6579}},
			{"node E", 45, []float64{816.7041}},
			{"node F", 45, []float64{1035.1096}},
		},
		styles: map[string]wantStyle{
			"node A":       {fill: "", stroke: "", text: "", width: 2, dashes: []float64{40, 9, 31, 12, 31, 8, 34, 8, 32}},
			"node B":       {fill: "", stroke: "", text: "", width: 3, dashes: []float64{22, 254.0982, 54.9297, 253.7826, 57.8596, 254.1274, 52.8187, 254.1274, 52.8187}},
			"node C":       {fill: "", stroke: "", text: "", width: 2, dashes: []float64{180, 317.6244, 31, 319.6244, 31, 315.9036, 31.0231, 315.9036, 31.0231}},
			"node D":       {fill: "", stroke: "", text: "", width: 2, dashes: []float64{114, 505.8101, 31.8944, 507.5989, 35.4721, 509.6129, 31, 506.3629, 31}},
			"node E":       {fill: "", stroke: "", text: "", width: 1, dashes: []float64{86, 745.1129, 99, 747.1129, 99, 743.0655, 96.2922, 744.0578, 98.0287}},
			"node F":       {fill: "", stroke: "", text: "", width: 2, dashes: []float64{28, 967.6128, 31, 971.6128, 31, 962.6128, 42, 962.6128, 38}},
			"edge L_A_B_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
			"edge L_B_C_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
			"edge L_C_D_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
			"edge L_D_E_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
			"edge L_E_F_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
		},
	},
	{
		name: "line styles", src: "flowchart LR\n  A --> B\n  A --o C\n  A --x D\n  A -.-> E\n  A --- F\n  A ~~~ G\n  A ~~~ H\n  A <--> I\n  A -->|l| J\n  linkStyle 0 stroke:#f00,stroke:#0f0\n  linkStyle 1 stroke:#16a34a,stroke-width:3px\n  linkStyle 2 Stroke:#2563eb\n  linkStyle 3 stroke-dasharray:5 5\n  linkStyle 4 fill:#ff08,stroke-dasharray:none\n  linkStyle 5 stroke-width:3\n  linkStyle 6 stroke-width:2px,stroke:red\n  linkStyle 7 stroke:#f00\n  linkStyle 8 color:#dc2626",
		size: [2]float64{612, 729.5},
		edges: map[string]wantEdge{
			"L_A_B_0": {box: [4]float64{160, 30.5, 96, 330}, width: 2, solids: []float64{160, 360.5, 162, 360.5, 250.0003, 30.5, 252.0003, 30.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{245.5, 24.1087, 256, 30.5, 245.5, 36.8913}, rings: []float64{}, cross: 0},
			"L_A_C_0": {box: [4]float64{160, 115.5, 100, 247.5}, width: 3, solids: []float64{160, 363, 162, 363, 245.5003, 115.5, 247.5003, 115.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{249.85, 115.5, 7}, cross: 0},
			"L_A_D_0": {box: [4]float64{160, 200.5, 100, 165}, width: 2, solids: []float64{160, 365.5, 162, 365.5, 245.5003, 200.5, 247.5003, 200.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{246.64, 195.3, 257.04, 205.7, 246.64, 205.7, 257.04, 195.3}, rings: []float64{}, cross: 2},
			"L_A_E_0": {box: [4]float64{160, 285.5, 96, 82.5}, width: 2, solids: []float64{}, hops: []float64{}, dots: []float64{},
				marks: []float64{245.5, 279.1087, 256, 285.5, 245.5, 291.8913}, rings: []float64{}, cross: 0},
			"L_A_F_0": {box: [4]float64{160, 370.5, 100, 0}, width: 2, solids: []float64{160, 370.5, 162, 370.5, 258, 370.5, 260, 370.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_A_G_0": {box: [4]float64{160, 373, 100, 82.5}, width: 0, solids: []float64{}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_A_H_0": {box: [4]float64{160, 375.5, 100, 165}, width: 2, solids: []float64{160, 375.5, 162, 375.5, 258, 540.5, 260, 540.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_A_I_0": {box: [4]float64{164, 378, 92, 247.5}, width: 2, solids: []float64{168, 378, 170, 378, 250.0002, 625.5, 252.0002, 625.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{163, 378, 174.5, 385, 174.5, 371, 245.5, 619.1087, 256, 625.5, 245.5, 631.8913}, rings: []float64{}, cross: 0},
			"L_A_J_0": {box: [4]float64{160, 380.5, 288, 318.5}, width: 2, solids: []float64{160, 380.5, 162, 380.5, 442.0002, 699, 444.0002, 699}, hops: []float64{}, dots: []float64{},
				marks: []float64{437.5, 692.6087, 448, 699, 437.5, 705.3913}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{},
		labels: map[string][2]float64{
			"L_A_J_0": {336, 698.5},
		},
		texts: []wantText{
			{"node A", 362, []float64{79.3281}},
			{"node B", 22, []float64{331.3281}},
			{"node C", 107, []float64{330.9375}},
			{"node D", 192, []float64{330.9375}},
			{"node E", 277, []float64{331.3281}},
			{"node F", 362, []float64{331.7188}},
			{"node G", 447, []float64{330.5469}},
			{"node H", 532, []float64{330.9375}},
			{"node I", 617, []float64{334.0469}},
			{"node J", 690.5, []float64{524.5}},
			{"label L_A_J_0", 690, []float64{334.4375}},
		},
		styles: map[string]wantStyle{
			"node A":        {fill: "", stroke: "", text: "", width: 2},
			"node B":        {fill: "", stroke: "", text: "", width: 2},
			"node C":        {fill: "", stroke: "", text: "", width: 2},
			"node D":        {fill: "", stroke: "", text: "", width: 2},
			"node E":        {fill: "", stroke: "", text: "", width: 2},
			"node F":        {fill: "", stroke: "", text: "", width: 2},
			"node G":        {fill: "", stroke: "", text: "", width: 2},
			"node H":        {fill: "", stroke: "", text: "", width: 2},
			"node I":        {fill: "", stroke: "", text: "", width: 2},
			"node J":        {fill: "", stroke: "", text: "", width: 2},
			"edge L_A_B_0":  {fill: "#00000000", stroke: "#00ff00ff", text: "", width: 2, marks: []string{"#ff0000ff"}},
			"edge L_A_C_0":  {fill: "#00000000", stroke: "#16a34aff", text: "", width: 3, marks: []string{""}},
			"edge L_A_D_0":  {fill: "#00000000", stroke: "#2563ebff", text: "", width: 2, marks: []string{""}},
			"edge L_A_E_0":  {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}, dashes: []float64{18, 161, 368, 164, 368, 253.8283, 285.5, 255, 285.5}},
			"edge L_A_F_0":  {fill: "#ffff0088", stroke: "", text: "", width: 2},
			"edge L_A_G_0":  {fill: "#00000000", stroke: "#00000000", text: "", width: 0},
			"edge L_A_H_0":  {fill: "#00000000", stroke: "#ff0000ff", text: "", width: 2},
			"edge L_A_I_0":  {fill: "#00000000", stroke: "#ff0000ff", text: "", width: 2, marks: []string{"", "#ff0000ff"}},
			"edge L_A_J_0":  {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
			"label L_A_J_0": {text: "#dc2626ff"},
		},
	},
	{
		name: "a color in capitals", src: "flowchart LR\n  A --> B --> C\n  D e1@-->|l| E\n  subgraph s [T]\n    F\n  end\n  subgraph t [U]\n    G\n  end\n  style A Color:#f00\n  classDef c COLOR:#0a0\n  class B,s c\n  class e1 c\n  linkStyle 0 Color:#f00\n  style s fill:#eee,stroke:#444\n  style t fill:#eee,stroke:#444,Color:#f00",
		size: [2]float64{600, 460},
		edges: map[string]wantEdge{
			"L_A_B_0": {box: [4]float64{208, 344.5, 36, 0}, width: 2, solids: []float64{208, 344.5, 210, 344.5, 238, 344.5, 240, 344.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{233.5, 338.1087, 244, 344.5, 233.5, 350.8913}, rings: []float64{}, cross: 0},
			"L_B_C_0": {box: [4]float64{400, 344.5, 36, 0}, width: 2, solids: []float64{400, 344.5, 402, 344.5, 430, 344.5, 432, 344.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{425.5, 338.1087, 436, 344.5, 425.5, 350.8913}, rings: []float64{}, cross: 0},
			"e1": {box: [4]float64{208, 429.5, 228, 0}, width: 2, solids: []float64{208, 429.5, 210, 429.5, 430, 429.5, 432, 429.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{425.5, 423.1087, 436, 429.5, 425.5, 435.8913}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{
			"t": {8, 8, 200, 117},
			"s": {8, 165, 200, 117},
		},
		labels: map[string][2]float64{
			"e1": {324, 429},
		},
		texts: []wantText{
			{"node A", 336, []float64{127.3281}},
			{"node B", 336, []float64{319.3281}},
			{"node C", 336, []float64{510.9375}},
			{"node D", 421, []float64{126.9375}},
			{"node E", 421, []float64{511.3281}},
			{"node F", 227, []float64{103.7188}},
			{"node G", 70, []float64{102.5469}},
			{"label e1", 420.5, []float64{322.4375}},
			{"group t", 10, []float64{102.9375}},
			{"group s", 167, []float64{103.7188}},
		},
		styles: map[string]wantStyle{
			"node A":       {fill: "", stroke: "", text: "", width: 2},
			"node B":       {fill: "", stroke: "", text: "#00aa00ff", width: 2},
			"node C":       {fill: "", stroke: "", text: "", width: 2},
			"node D":       {fill: "", stroke: "", text: "", width: 2},
			"node E":       {fill: "", stroke: "", text: "", width: 2},
			"node F":       {fill: "", stroke: "", text: "", width: 2},
			"node G":       {fill: "", stroke: "", text: "", width: 2},
			"group t":      {fill: "#eeeeeeff", stroke: "#444444ff", text: "on", width: 1},
			"group s":      {fill: "#eeeeeeff", stroke: "#444444ff", text: "on", width: 1},
			"edge L_A_B_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
			"edge L_B_C_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
			"edge e1":      {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
			"label e1":     {text: ""},
		},
	},
	{
		name: "a point twice in a row", src: "flowchart TD\n  n0[\\\\marker/]\n  n1{{arrow an}}\n  n1 -. yes .-> n0\n  n0 o--o|marker| n1\n  n1 --- n2\n  n4 o--o n0\n  n2 --- n0\n  n2 ==> n4\n  n5 --- n2\n  n0 o--o|no| n1\n  subgraph s1 [\"`**Bold** group title that is long`\"]\n    n0\n    n1\n    subgraph s2\n      n2\n    end\n  end",
		size: [2]float64{372.3133, 839},
		edges: map[string]wantEdge{
			"L_n1_n0_0": {box: [4]float64{110.36, 484, 5.4, 137}, width: 2, solids: []float64{110.36, 488.3655, 110.36, 486.3655, 110.36, 486.3655, 110.36, 484.3655}, hops: []float64{}, dots: []float64{34, 115.76, 620, 110.36, 491.3655},
				marks: []float64{103.9687, 494.5, 110.36, 484, 116.7513, 494.5}, rings: []float64{}, cross: 0},
			"L_n0_n1_0": {box: [4]float64{152.24, 480, 0, 141}, width: 2, solids: []float64{152.24, 492.5, 152.24, 494.5, 152.24, 606.5, 152.24, 608.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{152.24, 489.8, 7, 152.24, 610.85, 7}, cross: 0},
			"L_n1_n2_0": {box: [4]float64{44.68, 202, 173.9, 419}, width: 2, solids: []float64{79.28, 621, 79.28, 619, 218.58, 288, 218.58, 290}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_n4_n0_0": {box: [4]float64{236, 480, 16.34, 306}, width: 2, solids: []float64{252.34, 773.5, 252.34, 771.5, 236, 494.4997, 236, 492.4997}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{252.34, 776.2, 7, 236, 490.15, 7}, cross: 0},
			"L_n2_n0_0": {box: [4]float64{173.18, 335, 45.4, 94}, width: 2, solids: []float64{218.58, 335, 218.58, 337, 173.18, 427, 173.18, 429}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_n2_n4_0": {box: [4]float64{262.0467, 335, 34.2933, 447}, width: 3.5, solids: []float64{262.0467, 335, 262.0467, 337, 296.34, 776, 296.34, 778}, hops: []float64{}, dots: []float64{},
				marks: []float64{302.7313, 771.5, 296.34, 782, 289.9487, 771.5}, rings: []float64{}, cross: 0},
			"L_n5_n2_0": {box: [4]float64{262.0467, 53, 0, 237}, width: 2, solids: []float64{262.0467, 53, 262.0467, 55, 262.0467, 288, 262.0467, 290}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_n0_n1_2": {box: [4]float64{188.72, 480, 5.4, 141}, width: 2, solids: []float64{194.12, 492.5, 194.12, 494.5, 188.72, 606.5005, 188.72, 608.5005}, hops: []float64{}, dots: []float64{},
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
	{
		name: "lines hop", src: "flowchart TD\n  a0 & a1 & a2 --> b0 & b1 & b2",
		size: [2]float64{552, 226},
		edges: map[string]wantEdge{
			"L_a0_b0_0": {box: [4]float64{51, 53, 0, 116}, width: 2, solids: []float64{51, 53, 51, 55, 51, 163, 51, 165}, hops: []float64{}, dots: []float64{},
				marks: []float64{57.3913, 158.5, 51, 169, 44.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a0_b1_0": {box: [4]float64{84, 53, 159, 116}, width: 2, solids: []float64{84, 53, 84, 55, 243, 162.9998, 243, 164.9998}, hops: []float64{94.5, 133, 100.5, 127, 106.5, 133, 96.2574, 128.7574, 111, 133, 117, 127, 123, 133, 112.7574, 128.7574}, dots: []float64{},
				marks: []float64{249.3913, 158.5, 243, 169, 236.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a0_b2_0": {box: [4]float64{117, 53, 318, 116}, width: 2, solids: []float64{117, 53, 117, 55, 435, 162.9995, 435, 164.9995}, hops: []float64{237, 73, 243, 67, 249, 73, 238.7574, 68.7574, 270, 153, 276, 147, 282, 153, 271.7574, 148.7574, 303, 153, 309, 147, 315, 153, 304.7574, 148.7574}, dots: []float64{},
				marks: []float64{441.3913, 158.5, 435, 169, 428.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a1_b0_0": {box: [4]float64{84, 53, 159, 116}, width: 2, solids: []float64{243, 53, 243, 55, 84, 162.9997, 84, 164.9997}, hops: []float64{}, dots: []float64{},
				marks: []float64{90.3913, 158.5, 84, 169, 77.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a1_b1_0": {box: [4]float64{276, 53, 0, 116}, width: 2, solids: []float64{276, 53, 276, 55, 276, 163, 276, 165}, hops: []float64{}, dots: []float64{},
				marks: []float64{282.3913, 158.5, 276, 169, 269.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a1_b2_0": {box: [4]float64{309, 53, 159, 116}, width: 2, solids: []float64{309, 53, 309, 55, 468, 163.0002, 468, 165.0002}, hops: []float64{429, 73, 435, 67, 441, 73, 430.7574, 68.7574}, dots: []float64{},
				marks: []float64{474.3913, 158.5, 468, 169, 461.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a2_b0_0": {box: [4]float64{117, 53, 318, 116}, width: 2, solids: []float64{435, 53, 435, 55, 117, 162.9997, 117, 164.9997}, hops: []float64{282, 113, 276, 107, 270, 113, 280.2426, 108.7574, 265.5, 113, 259.5, 107, 253.5, 113, 263.7426, 108.7574}, dots: []float64{},
				marks: []float64{123.3913, 158.5, 117, 169, 110.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a2_b1_0": {box: [4]float64{309, 53, 159, 116}, width: 2, solids: []float64{468, 53, 468, 55, 309, 162.9999, 309, 164.9999}, hops: []float64{457.5, 133, 451.5, 127, 445.5, 133, 455.7426, 128.7574}, dots: []float64{},
				marks: []float64{315.3913, 158.5, 309, 169, 302.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a2_b2_0": {box: [4]float64{501, 53, 0, 116}, width: 2, solids: []float64{501, 53, 501, 55, 501, 163, 501, 165}, hops: []float64{}, dots: []float64{},
				marks: []float64{507.3913, 158.5, 501, 169, 494.6087, 158.5}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{},
		labels: map[string][2]float64{},
		texts: []wantText{
			{"node a0", 22, []float64{76.2031}},
			{"node a1", 22, []float64{268.2031}},
			{"node a2", 22, []float64{460.2031}},
			{"node b0", 187, []float64{76.2031}},
			{"node b1", 187, []float64{268.2031}},
			{"node b2", 187, []float64{460.2031}},
		},
	},
	{
		name: "hops side by side", src: "flowchart TD\n  a --> b & c & d & e\n  f --> b & c & d & e\n  g --> b & c & d & e\n  h --> b & c & d & e",
		size: [2]float64{744, 366},
		edges: map[string]wantEdge{
			"L_a_b_0": {box: [4]float64{44.4, 53, 0, 256}, width: 2, solids: []float64{44.4, 53, 44.4, 55, 44.4, 303, 44.4, 305}, hops: []float64{}, dots: []float64{},
				marks: []float64{50.7913, 298.5, 44.4, 309, 38.0087, 298.5}, rings: []float64{}, cross: 0},
			"L_a_c_0": {box: [4]float64{70.8, 53, 165.6, 256}, width: 2, solids: []float64{70.8, 53, 70.8, 55, 236.4, 303.0005, 236.4, 305.0005}, hops: []float64{79.871, 253, 84, 248.8767, 88.129, 253, 81.0804, 250.0844, 91.2, 253, 97.2, 247, 103.2, 253, 92.9574, 248.7574, 117.6, 253, 123.6, 247, 129.6, 253, 119.3574, 248.7574}, dots: []float64{},
				marks: []float64{242.7913, 298.5, 236.4, 309, 230.0087, 298.5}, rings: []float64{}, cross: 0},
			"L_a_d_0": {box: [4]float64{97.2, 53, 331.2, 256}, width: 2, solids: []float64{97.2, 53, 97.2, 55, 428.4, 303.0003, 428.4, 305.0003}, hops: []float64{117.6, 93, 123.6, 87, 129.6, 93, 119.3574, 88.7574, 258.671, 253, 262.8, 248.8791, 266.929, 253, 259.8804, 250.0861, 283.2, 253, 289.2, 247, 295.2, 253, 284.9574, 248.7574, 309.6, 253, 315.6, 247, 321.6, 253, 311.3574, 248.7574}, dots: []float64{},
				marks: []float64{434.7913, 298.5, 428.4, 309, 422.0087, 298.5}, rings: []float64{}, cross: 0},
			"L_a_e_0": {box: [4]float64{123.6, 53, 496.8, 256}, width: 2, solids: []float64{123.6, 53, 123.6, 55, 620.4, 302.9997, 620.4, 304.9997}, hops: []float64{243.6, 213, 249.6, 207, 255.6, 213, 245.3574, 208.7574, 256.8, 213, 262.8, 207, 268.8, 213, 258.5574, 208.7574, 283.2, 213, 289.2, 207, 295.2, 213, 284.9574, 208.7574, 309.6, 213, 315.6, 207, 321.6, 213, 311.3574, 208.7574, 435.6, 213, 441.6, 207, 447.6, 213, 437.3574, 208.7574, 475.2, 253, 481.2, 247, 487.2, 253, 476.9574, 248.7574}, dots: []float64{},
				marks: []float64{626.7913, 298.5, 620.4, 309, 614.0087, 298.5}, rings: []float64{}, cross: 0},
			"L_f_b_0": {box: [4]float64{70.8, 53, 165.6, 256}, width: 2, solids: []float64{236.4, 53, 236.4, 55, 70.8, 302.9999, 70.8, 304.9999}, hops: []float64{129.6, 73, 123.6, 67, 117.6, 73, 127.8426, 68.7574, 101.329, 73, 97.2, 68.871, 93.071, 73, 100.1196, 70.0804}, dots: []float64{},
				marks: []float64{77.1913, 298.5, 70.8, 309, 64.4087, 298.5}, rings: []float64{}, cross: 0},
			"L_f_c_0": {box: [4]float64{262.8, 53, 0, 256}, width: 2, solids: []float64{262.8, 53, 262.8, 55, 262.8, 303, 262.8, 305}, hops: []float64{}, dots: []float64{},
				marks: []float64{269.1913, 298.5, 262.8, 309, 256.4087, 298.5}, rings: []float64{}, cross: 0},
			"L_f_d_0": {box: [4]float64{289.2, 53, 165.6, 256}, width: 2, solids: []float64{289.2, 53, 289.2, 55, 454.8, 302.9999, 454.8, 304.9999}, hops: []float64{424.271, 133, 428.4, 128.871, 432.529, 133, 425.4804, 130.0803}, dots: []float64{},
				marks: []float64{461.1913, 298.5, 454.8, 309, 448.4087, 298.5}, rings: []float64{}, cross: 0},
			"L_f_e_0": {box: [4]float64{315.6, 53, 331.2, 256}, width: 2, solids: []float64{315.6, 53, 315.6, 55, 646.8, 302.9998, 646.8, 304.9998}, hops: []float64{422.4, 113, 428.4, 107, 434.4, 113, 424.1574, 108.7574, 448.8, 113, 454.8, 107, 460.8, 113, 450.5574, 108.7574, 462, 113, 468, 107, 474, 113, 463.7574, 108.7574, 475.2, 113, 481.2, 107, 487.2, 113, 476.9574, 108.7574, 614.4, 113, 620.4, 107, 626.4, 113, 616.1574, 108.7574}, dots: []float64{},
				marks: []float64{653.1913, 298.5, 646.8, 309, 640.4087, 298.5}, rings: []float64{}, cross: 0},
			"L_g_b_0": {box: [4]float64{97.2, 53, 331.2, 256}, width: 2, solids: []float64{428.4, 53, 428.4, 55, 97.2, 303.0003, 97.2, 305.0003}, hops: []float64{268.8, 153, 262.8, 147, 256.8, 153, 267.0426, 148.7574, 255.6, 153, 249.6, 147, 243.6, 153, 253.8426, 148.7574, 129.6, 153, 123.6, 147, 117.6, 153, 127.8426, 148.7574}, dots: []float64{},
				marks: []float64{103.5913, 298.5, 97.2, 309, 90.8087, 298.5}, rings: []float64{}, cross: 0},
			"L_g_c_0": {box: [4]float64{289.2, 53, 165.6, 256}, width: 2, solids: []float64{454.8, 53, 454.8, 55, 289.2, 302.9997, 289.2, 304.9997}, hops: []float64{445.729, 173, 441.6, 168.871, 437.471, 173, 444.5196, 170.0804}, dots: []float64{},
				marks: []float64{295.5913, 298.5, 289.2, 309, 282.8087, 298.5}, rings: []float64{}, cross: 0},
			"L_g_d_0": {box: [4]float64{481.2, 53, 0, 256}, width: 2, solids: []float64{481.2, 53, 481.2, 55, 481.2, 303, 481.2, 305}, hops: []float64{}, dots: []float64{},
				marks: []float64{487.5913, 298.5, 481.2, 309, 474.8087, 298.5}, rings: []float64{}, cross: 0},
			"L_g_e_0": {box: [4]float64{507.6, 53, 165.6, 256}, width: 2, solids: []float64{507.6, 53, 507.6, 55, 673.2, 302.9996, 673.2, 304.9996}, hops: []float64{614.4, 73, 620.4, 67, 626.4, 73, 616.1574, 68.7574, 640.8, 73, 646.8, 67, 652.8, 73, 642.5574, 68.7574}, dots: []float64{},
				marks: []float64{679.5913, 298.5, 673.2, 309, 666.8087, 298.5}, rings: []float64{}, cross: 0},
			"L_h_b_0": {box: [4]float64{123.6, 53, 496.8, 256}, width: 2, solids: []float64{620.4, 53, 620.4, 55, 123.6, 303, 123.6, 305}, hops: []float64{487.2, 233, 481.2, 227, 475.2, 233, 485.4426, 228.7574, 467.4, 233, 461.4, 227, 455.4, 233, 465.6426, 228.7574, 447.6, 233, 441.6, 227, 435.6, 233, 445.8426, 228.7574, 321.6, 233, 315.6, 227, 309.6, 233, 319.8426, 228.7574, 295.2, 233, 289.2, 227, 283.2, 233, 293.4426, 228.7574, 268.8, 233, 262.8, 227, 256.8, 233, 267.0426, 228.7574, 255.6, 233, 249.6, 227, 243.6, 233, 253.8426, 228.7574}, dots: []float64{},
				marks: []float64{129.9913, 298.5, 123.6, 309, 117.2087, 298.5}, rings: []float64{}, cross: 0},
			"L_h_c_0": {box: [4]float64{315.6, 53, 331.2, 256}, width: 2, solids: []float64{646.8, 53, 646.8, 55, 315.6, 303.0003, 315.6, 305.0003}, hops: []float64{626.4, 93, 620.4, 87, 614.4, 93, 624.6426, 88.7574, 485.329, 93, 481.2, 88.871, 477.071, 93, 484.1196, 90.0804, 447.6, 193, 441.6, 187, 435.6, 193, 445.8426, 188.7574}, dots: []float64{},
				marks: []float64{321.9913, 298.5, 315.6, 309, 309.2087, 298.5}, rings: []float64{}, cross: 0},
			"L_h_d_0": {box: [4]float64{507.6, 53, 165.6, 256}, width: 2, solids: []float64{673.2, 53, 673.2, 55, 507.6, 302.9996, 507.6, 304.9996}, hops: []float64{652.8, 273, 646.8, 267, 640.8, 273, 651.0426, 268.7574, 626.4, 273, 620.4, 267, 614.4, 273, 624.6426, 268.7574}, dots: []float64{},
				marks: []float64{513.9913, 298.5, 507.6, 309, 501.2087, 298.5}, rings: []float64{}, cross: 0},
			"L_h_e_0": {box: [4]float64{699.6, 53, 0, 256}, width: 2, solids: []float64{699.6, 53, 699.6, 55, 699.6, 303, 699.6, 305}, hops: []float64{}, dots: []float64{},
				marks: []float64{705.9913, 298.5, 699.6, 309, 693.2087, 298.5}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{},
		labels: map[string][2]float64{},
		texts: []wantText{
			{"node a", 22, []float64{80.0938}},
			{"node b", 327, []float64{80.0938}},
			{"node c", 327, []float64{272.5}},
			{"node d", 327, []float64{464.0938}},
			{"node e", 327, []float64{656.0938}},
			{"node f", 22, []float64{274.0469}},
			{"node g", 22, []float64{464.0938}},
			{"node h", 22, []float64{656.0938}},
		},
	},
	{
		name: "hopping dotted lines go solid", src: "flowchart TD\n  a0 & a1 & a2 -.-> b0 & b1 & b2",
		size: [2]float64{552, 226},
		edges: map[string]wantEdge{
			"L_a0_b0_0": {box: [4]float64{51, 53, 0, 116}, width: 2, solids: []float64{51, 165, 51, 167, 51, 167, 51, 169}, hops: []float64{}, dots: []float64{28, 51, 54, 51, 162},
				marks: []float64{57.3913, 158.5, 51, 169, 44.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a0_b1_0": {box: [4]float64{84, 53, 159, 116}, width: 2, solids: []float64{84, 53, 84, 55, 243, 164.9998, 243, 166.9998}, hops: []float64{94.5, 133, 100.5, 127, 106.5, 133, 96.2574, 128.7574, 111, 133, 117, 127, 123, 133, 112.7574, 128.7574}, dots: []float64{},
				marks: []float64{249.3913, 158.5, 243, 169, 236.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a0_b2_0": {box: [4]float64{117, 53, 318, 116}, width: 2, solids: []float64{117, 53, 117, 55, 435, 164.9995, 435, 166.9995}, hops: []float64{237, 73, 243, 67, 249, 73, 238.7574, 68.7574, 270, 153, 276, 147, 282, 153, 271.7574, 148.7574, 303, 153, 309, 147, 315, 153, 304.7574, 148.7574}, dots: []float64{},
				marks: []float64{441.3913, 158.5, 435, 169, 428.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a1_b0_0": {box: [4]float64{84, 53, 159, 116}, width: 2, solids: []float64{84, 164.6567, 84, 166.6567, 84, 166.6567, 84, 168.6567}, hops: []float64{}, dots: []float64{65, 243, 54, 84, 161.6567},
				marks: []float64{90.3913, 158.5, 84, 169, 77.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a1_b1_0": {box: [4]float64{276, 53, 0, 116}, width: 2, solids: []float64{276, 165, 276, 167, 276, 167, 276, 169}, hops: []float64{}, dots: []float64{28, 276, 54, 276, 162},
				marks: []float64{282.3913, 158.5, 276, 169, 269.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a1_b2_0": {box: [4]float64{309, 53, 159, 116}, width: 2, solids: []float64{309, 53, 309, 55, 468, 165.0002, 468, 167.0002}, hops: []float64{429, 73, 435, 67, 441, 73, 430.7574, 68.7574}, dots: []float64{},
				marks: []float64{474.3913, 158.5, 468, 169, 461.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a2_b0_0": {box: [4]float64{117, 53, 318, 116}, width: 2, solids: []float64{435, 53, 435, 55, 117, 164.9997, 117, 166.9997}, hops: []float64{282, 113, 276, 107, 270, 113, 280.2426, 108.7574, 265.5, 113, 259.5, 107, 253.5, 113, 263.7426, 108.7574}, dots: []float64{},
				marks: []float64{123.3913, 158.5, 117, 169, 110.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a2_b1_0": {box: [4]float64{309, 53, 159, 116}, width: 2, solids: []float64{468, 53, 468, 55, 309, 164.9999, 309, 166.9999}, hops: []float64{457.5, 133, 451.5, 127, 445.5, 133, 455.7426, 128.7574}, dots: []float64{},
				marks: []float64{315.3913, 158.5, 309, 169, 302.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a2_b2_0": {box: [4]float64{501, 53, 0, 116}, width: 2, solids: []float64{501, 165, 501, 167, 501, 167, 501, 169}, hops: []float64{}, dots: []float64{28, 501, 54, 501, 162},
				marks: []float64{507.3913, 158.5, 501, 169, 494.6087, 158.5}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{},
		labels: map[string][2]float64{},
		texts: []wantText{
			{"node a0", 22, []float64{76.2031}},
			{"node a1", 22, []float64{268.2031}},
			{"node a2", 22, []float64{460.2031}},
			{"node b0", 187, []float64{76.2031}},
			{"node b1", 187, []float64{268.2031}},
			{"node b2", 187, []float64{460.2031}},
		},
	},
	{
		name: "hopping lines lose their author's dashes", src: "flowchart TD\n  a0 & a1 & a2 --> b0 & b1 & b2\n  linkStyle 0,2,4,6,8 stroke-dasharray:5 5,stroke:#f00\n  linkStyle 1,3,5,7 stroke-dasharray:none",
		size: [2]float64{552, 226},
		edges: map[string]wantEdge{
			"L_a0_b0_0": {box: [4]float64{51, 53, 0, 116}, width: 2, solids: []float64{}, hops: []float64{}, dots: []float64{},
				marks: []float64{57.3913, 158.5, 51, 169, 44.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a0_b1_0": {box: [4]float64{84, 53, 159, 116}, width: 2, solids: []float64{84, 53, 84, 55, 243, 162.9998, 243, 164.9998}, hops: []float64{94.5, 133, 100.5, 127, 106.5, 133, 96.2574, 128.7574, 111, 133, 117, 127, 123, 133, 112.7574, 128.7574}, dots: []float64{},
				marks: []float64{249.3913, 158.5, 243, 169, 236.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a0_b2_0": {box: [4]float64{117, 53, 318, 116}, width: 2, solids: []float64{117, 53, 117, 55, 435, 162.9995, 435, 164.9995}, hops: []float64{237, 73, 243, 67, 249, 73, 238.7574, 68.7574, 270, 153, 276, 147, 282, 153, 271.7574, 148.7574, 303, 153, 309, 147, 315, 153, 304.7574, 148.7574}, dots: []float64{},
				marks: []float64{441.3913, 158.5, 435, 169, 428.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a1_b0_0": {box: [4]float64{84, 53, 159, 116}, width: 2, solids: []float64{243, 53, 243, 55, 84, 167, 84, 169}, hops: []float64{}, dots: []float64{},
				marks: []float64{90.3913, 158.5, 84, 169, 77.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a1_b1_0": {box: [4]float64{276, 53, 0, 116}, width: 2, solids: []float64{}, hops: []float64{}, dots: []float64{},
				marks: []float64{282.3913, 158.5, 276, 169, 269.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a1_b2_0": {box: [4]float64{309, 53, 159, 116}, width: 2, solids: []float64{309, 53, 309, 55, 468, 163.0002, 468, 165.0002}, hops: []float64{429, 73, 435, 67, 441, 73, 430.7574, 68.7574}, dots: []float64{},
				marks: []float64{474.3913, 158.5, 468, 169, 461.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a2_b0_0": {box: [4]float64{117, 53, 318, 116}, width: 2, solids: []float64{435, 53, 435, 55, 117, 162.9997, 117, 164.9997}, hops: []float64{282, 113, 276, 107, 270, 113, 280.2426, 108.7574, 265.5, 113, 259.5, 107, 253.5, 113, 263.7426, 108.7574}, dots: []float64{},
				marks: []float64{123.3913, 158.5, 117, 169, 110.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a2_b1_0": {box: [4]float64{309, 53, 159, 116}, width: 2, solids: []float64{468, 53, 468, 55, 309, 162.9999, 309, 164.9999}, hops: []float64{457.5, 133, 451.5, 127, 445.5, 133, 455.7426, 128.7574}, dots: []float64{},
				marks: []float64{315.3913, 158.5, 309, 169, 302.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a2_b2_0": {box: [4]float64{501, 53, 0, 116}, width: 2, solids: []float64{}, hops: []float64{}, dots: []float64{},
				marks: []float64{507.3913, 158.5, 501, 169, 494.6087, 158.5}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{},
		labels: map[string][2]float64{},
		texts: []wantText{
			{"node a0", 22, []float64{76.2031}},
			{"node a1", 22, []float64{268.2031}},
			{"node a2", 22, []float64{460.2031}},
			{"node b0", 187, []float64{76.2031}},
			{"node b1", 187, []float64{268.2031}},
			{"node b2", 187, []float64{460.2031}},
		},
		styles: map[string]wantStyle{
			"node a0":        {fill: "", stroke: "", text: "", width: 2},
			"node a1":        {fill: "", stroke: "", text: "", width: 2},
			"node a2":        {fill: "", stroke: "", text: "", width: 2},
			"node b0":        {fill: "", stroke: "", text: "", width: 2},
			"node b1":        {fill: "", stroke: "", text: "", width: 2},
			"node b2":        {fill: "", stroke: "", text: "", width: 2},
			"edge L_a0_b0_0": {fill: "#00000000", stroke: "#ff0000ff", text: "", width: 2, marks: []string{"#ff0000ff"}, dashes: []float64{12, 51, 54, 51, 57, 51, 164, 51, 167}},
			"edge L_a0_b1_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
			"edge L_a0_b2_0": {fill: "#00000000", stroke: "#ff0000ff", text: "", width: 2, marks: []string{"#ff0000ff"}},
			"edge L_a1_b0_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
			"edge L_a1_b1_0": {fill: "#00000000", stroke: "#ff0000ff", text: "", width: 2, marks: []string{"#ff0000ff"}, dashes: []float64{12, 276, 54, 276, 57, 276, 164, 276, 167}},
			"edge L_a1_b2_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
			"edge L_a2_b0_0": {fill: "#00000000", stroke: "#ff0000ff", text: "", width: 2, marks: []string{"#ff0000ff"}},
			"edge L_a2_b1_0": {fill: "#00000000", stroke: "", text: "", width: 2, marks: []string{""}},
			"edge L_a2_b2_0": {fill: "#00000000", stroke: "#ff0000ff", text: "", width: 2, marks: []string{"#ff0000ff"}, dashes: []float64{12, 501, 54, 501, 57, 501, 164, 501, 167}},
		},
	},
	{
		name: "invisible lines cross too", src: "flowchart TD\n  a0 ~~~ b0\n  a0 --> b1\n  a0 --> b2\n  a1 --> b0\n  a1 ~~~ b1\n  a1 --> b2\n  a2 --> b0\n  a2 --> b1\n  a2 ~~~ b2",
		size: [2]float64{552, 226},
		edges: map[string]wantEdge{
			"L_a0_b0_0": {box: [4]float64{51, 53, 0, 120}, width: 0, solids: []float64{}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_a0_b1_0": {box: [4]float64{84, 53, 159, 116}, width: 2, solids: []float64{84, 53, 84, 55, 243, 162.9998, 243, 164.9998}, hops: []float64{94.5, 133, 100.5, 127, 106.5, 133, 96.2574, 128.7574, 111, 133, 117, 127, 123, 133, 112.7574, 128.7574}, dots: []float64{},
				marks: []float64{249.3913, 158.5, 243, 169, 236.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a0_b2_0": {box: [4]float64{117, 53, 318, 116}, width: 2, solids: []float64{117, 53, 117, 55, 435, 162.9995, 435, 164.9995}, hops: []float64{237, 73, 243, 67, 249, 73, 238.7574, 68.7574, 270, 153, 276, 147, 282, 153, 271.7574, 148.7574, 303, 153, 309, 147, 315, 153, 304.7574, 148.7574}, dots: []float64{},
				marks: []float64{441.3913, 158.5, 435, 169, 428.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a1_b0_0": {box: [4]float64{84, 53, 159, 116}, width: 2, solids: []float64{243, 53, 243, 55, 84, 162.9997, 84, 164.9997}, hops: []float64{}, dots: []float64{},
				marks: []float64{90.3913, 158.5, 84, 169, 77.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a1_b1_0": {box: [4]float64{276, 53, 0, 120}, width: 0, solids: []float64{}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_a1_b2_0": {box: [4]float64{309, 53, 159, 116}, width: 2, solids: []float64{309, 53, 309, 55, 468, 163.0002, 468, 165.0002}, hops: []float64{429, 73, 435, 67, 441, 73, 430.7574, 68.7574}, dots: []float64{},
				marks: []float64{474.3913, 158.5, 468, 169, 461.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a2_b0_0": {box: [4]float64{117, 53, 318, 116}, width: 2, solids: []float64{435, 53, 435, 55, 117, 162.9997, 117, 164.9997}, hops: []float64{282, 113, 276, 107, 270, 113, 280.2426, 108.7574, 265.5, 113, 259.5, 107, 253.5, 113, 263.7426, 108.7574}, dots: []float64{},
				marks: []float64{123.3913, 158.5, 117, 169, 110.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a2_b1_0": {box: [4]float64{309, 53, 159, 116}, width: 2, solids: []float64{468, 53, 468, 55, 309, 162.9999, 309, 164.9999}, hops: []float64{457.5, 133, 451.5, 127, 445.5, 133, 455.7426, 128.7574}, dots: []float64{},
				marks: []float64{315.3913, 158.5, 309, 169, 302.6087, 158.5}, rings: []float64{}, cross: 0},
			"L_a2_b2_0": {box: [4]float64{501, 53, 0, 120}, width: 0, solids: []float64{}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{},
		labels: map[string][2]float64{},
		texts: []wantText{
			{"node a0", 22, []float64{76.2031}},
			{"node b0", 187, []float64{76.2031}},
			{"node b1", 187, []float64{268.2031}},
			{"node b2", 187, []float64{460.2031}},
			{"node a1", 22, []float64{268.2031}},
			{"node a2", 22, []float64{460.2031}},
		},
	},
	{
		name: "a dotted line with no room for its hop", src: "flowchart TD\n  a0 o--o b0\n  a0 -.-> b1\n  a1 -. lb .-> b0\n  a1 ~~~ b1\n  a2 -.- b0\n  a2 -.- b1",
		size: [2]float64{430.9062, 251},
		edges: map[string]wantEdge{
			"L_a0_b0_0": {box: [4]float64{112.9062, 138, 0, 60}, width: 2, solids: []float64{112.9062, 150.5, 112.9062, 152.5, 112.9062, 183.5, 112.9062, 185.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{112.9062, 147.8, 7, 112.9062, 187.85, 7}, cross: 0},
			"L_a0_b1_0": {box: [4]float64{156.9062, 138, 153.5, 56}, width: 2, solids: []float64{310.4062, 189.8283, 310.4062, 191.8283, 310.4062, 191.8283, 310.4062, 193.8283}, hops: []float64{}, dots: []float64{50, 156.9062, 139, 310.4062, 186.8283},
				marks: []float64{316.7976, 183.5, 310.4062, 194, 304.0149, 183.5}, rings: []float64{}, cross: 0},
			"L_a1_b0_0": {box: [4]float64{13.5, 53, 182.7031, 141}, width: 2, solids: []float64{79.9062, 186.5473, 79.9062, 188.5473, 79.9062, 188.5473, 79.9062, 190.5473, 79.9062, 192.5473, 79.9062, 194, 79.9062, 192.5473, 79.9062, 194}, hops: []float64{}, dots: []float64{93, 196.2031, 54, 79.9062, 183.5473},
				marks: []float64{86.2976, 183.5, 79.9062, 194, 73.5149, 183.5}, rings: []float64{}, cross: 0},
			"L_a1_b1_0": {box: [4]float64{240.2031, 53, 182.7031, 145}, width: 0, solids: []float64{}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_a2_b0_0": {box: [4]float64{145.906, 138, 159, 60}, width: 2, solids: []float64{304.906, 138, 304.906, 140, 145.906, 194.0002, 145.906, 196.0002}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_a2_b1_0": {box: [4]float64{343.4062, 138, 5.5, 60}, width: 2, solids: []float64{343.4062, 196.5722, 343.4062, 198, 343.4062, 196.5722, 343.4062, 198}, hops: []float64{}, dots: []float64{15, 348.9062, 139, 343.4062, 191.5722},
				marks: []float64{}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{},
		labels: map[string][2]float64{
			"L_a1_b0_0": {13.4531, 115.5},
		},
		texts: []wantText{
			{"node a0", 107, []float64{127.1094}},
			{"node b0", 212, []float64{105.1094}},
			{"node b1", 212, []float64{335.6094}},
			{"node a1", 22, []float64{210.4062}},
			{"node a2", 107, []float64{319.1094}},
			{"label L_a1_b0_0", 107, []float64{8}},
		},
	},
	{
		name: "hops too close for both", src: "flowchart TD\n  a0 --- b0\n  a0 --x b1\n  a0 -. lb .-> b2\n  a1 --o b0\n  a1 --o b1\n  a1 -. lb .-> b2\n  a2 --o b0\n  a2 --> b1\n  a2 ~~~ b2\n  a3 o--o b0\n  a3 --o b1\n  a3 --> b2",
		size: [2]float64{812.2031, 391},
		edges: map[string]wantEdge{
			"L_a0_b0_0": {box: [4]float64{256.753, 53, 212.597, 180}, width: 2, solids: []float64{469.35, 53, 469.35, 55, 256.753, 231, 256.753, 233}, hops: []float64{289.153, 213, 283.153, 207, 277.153, 213, 287.3956, 208.7574}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_a0_b1_0": {box: [4]float64{502.35, 53, 80, 180}, width: 2, solids: []float64{502.35, 53, 502.35, 55, 582.35, 218.5003, 582.35, 220.5003}, hops: []float64{}, dots: []float64{},
				marks: []float64{587.55, 219.64, 577.15, 230.04, 577.15, 219.64, 587.55, 230.04}, rings: []float64{}, cross: 2},
			"L_a0_b2_0": {box: [4]float64{419.55, 53, 271.1, 281}, width: 2, solids: []float64{419.55, 329.2565, 419.55, 331.2565, 419.55, 331.2565, 419.55, 333.2565}, hops: []float64{}, dots: []float64{173, 535.35, 54, 419.55, 326.2565},
				marks: []float64{425.9413, 323.5, 419.55, 334, 413.1587, 323.5}, rings: []float64{}, cross: 0},
			"L_a1_b0_0": {box: [4]float64{230.353, 53, 79.997, 180}, width: 2, solids: []float64{310.35, 53, 310.35, 55, 230.353, 218.4999, 230.353, 220.4999}, hops: []float64{289.153, 193, 283.153, 187, 277.153, 193, 287.3956, 188.7574}, dots: []float64{},
				marks: []float64{}, rings: []float64{230.353, 222.85, 7}, cross: 0},
			"L_a1_b1_0": {box: [4]float64{343.35, 53, 212.6, 180}, width: 2, solids: []float64{343.35, 53, 343.35, 55, 555.95, 218.4999, 555.95, 220.4999}, hops: []float64{463.35, 133, 469.35, 127, 475.35, 133, 465.1074, 128.7574}, dots: []float64{},
				marks: []float64{}, rings: []float64{555.95, 222.85, 7}, cross: 0},
			"L_a1_b2_0": {box: [4]float64{122.1469, 53, 271.0031, 281}, width: 2, solids: []float64{393.15, 329.4503, 393.15, 331.4503, 393.15, 331.4503, 393.15, 333.4503}, hops: []float64{}, dots: []float64{173, 277.35, 54, 393.15, 326.4503},
				marks: []float64{399.5413, 323.5, 393.15, 334, 386.7587, 323.5}, rings: []float64{}, cross: 0},
			"L_a2_b0_0": {box: [4]float64{84, 53, 119.953, 180}, width: 2, solids: []float64{84, 53, 84, 55, 203.953, 218.4999, 203.953, 220.4999}, hops: []float64{116.147, 193, 122.147, 187, 128.147, 193, 117.9044, 188.7574}, dots: []float64{},
				marks: []float64{}, rings: []float64{203.953, 222.85, 7}, cross: 0},
			"L_a2_b1_0": {box: [4]float64{117, 53, 412.55, 176}, width: 2, solids: []float64{117, 53, 117, 55, 529.55, 222.9998, 529.55, 224.9998}, hops: []float64{304.35, 153, 310.35, 147, 316.35, 153, 306.1074, 148.7574, 463.35, 153, 469.35, 147, 475.35, 153, 465.1074, 148.7574}, dots: []float64{},
				marks: []float64{535.9413, 218.5, 529.55, 229, 523.1587, 218.5}, rings: []float64{}, cross: 0},
			"L_a2_b2_0": {box: [4]float64{50.9969, 53, 315.7531, 285}, width: 0, solids: []float64{}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{}, cross: 0},
			"L_a3_b0_0": {box: [4]float64{283.153, 53, 412.05, 180}, width: 2, solids: []float64{695.203, 65.5, 695.1049, 67.4958, 283.153, 218.4998, 283.153, 220.4998}, hops: []float64{541.35, 73, 535.35, 67, 529.35, 73, 539.5926, 68.7574, 508.35, 73, 502.35, 67.0166, 496.35, 73, 506.5926, 68.7691, 475.35, 73, 469.35, 67, 463.35, 73, 473.5926, 68.7574, 349.35, 73, 343.35, 67, 337.35, 73, 347.5926, 68.7574, 316.35, 73, 310.35, 67, 304.35, 73, 314.5926, 68.7574}, dots: []float64{},
				marks: []float64{}, rings: []float64{695.203, 62.8, 7, 283.153, 222.85, 7}, cross: 0},
			"L_a3_b1_0": {box: [4]float64{608.75, 53, 119.453, 180}, width: 2, solids: []float64{728.203, 53, 728.203, 55, 608.75, 218.4999, 608.75, 220.4999}, hops: []float64{696.65, 113, 690.65, 107, 684.65, 113, 694.8926, 108.7574}, dots: []float64{},
				marks: []float64{}, rings: []float64{608.75, 222.85, 7}, cross: 0},
			"L_a3_b2_0": {box: [4]float64{445.95, 53, 315.2531, 281}, width: 2, solids: []float64{761.2031, 53, 761.2031, 55, 445.95, 328.0002, 445.95, 330.0002}, hops: []float64{}, dots: []float64{},
				marks: []float64{452.3413, 323.5, 445.95, 334, 439.5587, 323.5}, rings: []float64{}, cross: 0},
		},
		groups: map[string][4]float64{},
		labels: map[string][2]float64{
			"L_a0_b2_0": {690.6031, 255.5},
			"L_a1_b2_0": {122.1, 255.5},
		},
		texts: []wantText{
			{"node a0", 22, []float64{494.5531}},
			{"node b0", 247, []float64{235.7563}},
			{"node b1", 247, []float64{561.3531}},
			{"node b2", 352, []float64{398.5531}},
			{"node a1", 22, []float64{302.5531}},
			{"node a2", 22, []float64{76.2031}},
			{"node a3", 22, []float64{720.4062}},
			{"label L_a0_b2_0", 247, []float64{685.15}},
			{"label L_a1_b2_0", 247, []float64{116.6469}},
		},
	},
	{
		name: "hops with circles and crosses", src: "flowchart LR\n  a0 & a1 o--o b0 & b1 & b2\n  a2 x--x b0 & b1 & b2",
		size: [2]float64{480, 231},
		edges: map[string]wantEdge{
			"L_a0_b0_0": {box: [4]float64{160, 24.25, 160, 0}, width: 2, solids: []float64{172.5, 24.25, 174.5, 24.25, 305.5, 24.25, 307.5, 24.25}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{169.8, 24.25, 7, 309.85, 24.25, 7}, cross: 0},
			"L_a0_b1_0": {box: [4]float64{160, 30.5, 160, 78.75}, width: 2, solids: []float64{172.5, 30.5, 174.5, 30.5, 305.5003, 109.25, 307.5003, 109.25}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{169.8, 30.5, 7, 309.85, 109.25, 7}, cross: 0},
			"L_a0_b2_0": {box: [4]float64{160, 36.75, 160, 157.5}, width: 2, solids: []float64{172.5, 36.75, 174.5, 36.75, 305.5002, 194.25, 307.5002, 194.25}, hops: []float64{214, 112.375, 220, 106.375, 226, 112.375, 215.7574, 108.1324}, dots: []float64{},
				marks: []float64{}, rings: []float64{169.8, 36.75, 7, 309.85, 194.25, 7}, cross: 0},
			"L_a1_b0_0": {box: [4]float64{160, 30.5, 160, 78.75}, width: 2, solids: []float64{172.5, 109.25, 174.4958, 109.152, 305.5004, 30.5, 307.5004, 30.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{169.8, 109.25, 7, 309.85, 30.5, 7}, cross: 0},
			"L_a1_b1_0": {box: [4]float64{160, 109.5, 160, 6}, width: 2, solids: []float64{172.5, 115.5, 174.5, 115.5, 305.4997, 115.5, 307.4997, 115.5}, hops: []float64{214, 115.5, 220, 109.5, 226, 115.5, 215.7574, 111.2574}, dots: []float64{},
				marks: []float64{}, rings: []float64{169.8, 115.5, 7, 309.85, 115.5, 7}, cross: 0},
			"L_a1_b2_0": {box: [4]float64{160, 121.75, 160, 78.75}, width: 2, solids: []float64{172.5, 121.75, 174.4958, 121.848, 305.4997, 200.5, 307.4997, 200.5}, hops: []float64{}, dots: []float64{},
				marks: []float64{}, rings: []float64{169.8, 121.75, 7, 309.85, 200.5, 7}, cross: 0},
			"L_a2_b0_0": {box: [4]float64{160, 36.75, 160, 157.5}, width: 2, solids: []float64{172.5, 194.25, 174.5, 194.25, 305.5003, 36.75, 307.5003, 36.75}, hops: []float64{}, dots: []float64{},
				marks: []float64{163.6, 189.05, 174, 199.45, 163.6, 199.45, 174, 189.05, 306.64, 31.55, 317.04, 41.95, 306.64, 41.95, 317.04, 31.55}, rings: []float64{}, cross: 2},
			"L_a2_b1_0": {box: [4]float64{160, 115.75, 160, 84.75}, width: 2, solids: []float64{172.5, 200.5, 174.5, 200.5, 305.4997, 121.75, 307.4997, 121.75}, hops: []float64{214, 121.75, 220, 115.75, 226, 121.75, 215.7574, 117.5074, 254, 121.75, 260, 115.75, 266, 121.75, 255.7574, 117.5074}, dots: []float64{},
				marks: []float64{163.6, 195.3, 174, 205.7, 163.6, 205.7, 174, 195.3, 306.64, 116.55, 317.04, 126.95, 306.64, 126.95, 317.04, 116.55}, rings: []float64{}, cross: 2},
			"L_a2_b2_0": {box: [4]float64{160, 206.75, 160, 0}, width: 2, solids: []float64{172.5, 206.75, 174.5, 206.75, 305.5, 206.75, 307.5, 206.75}, hops: []float64{}, dots: []float64{},
				marks: []float64{163.6, 201.55, 174, 211.95, 163.6, 211.95, 174, 201.55, 306.64, 201.55, 317.04, 211.95, 306.64, 211.95, 317.04, 201.55}, rings: []float64{}, cross: 2},
		},
		groups: map[string][4]float64{},
		labels: map[string][2]float64{},
		texts: []wantText{
			{"node a0", 22, []float64{76.2031}},
			{"node a1", 107, []float64{76.2031}},
			{"node b0", 22, []float64{388.2031}},
			{"node b1", 107, []float64{388.2031}},
			{"node b2", 192, []float64{388.2031}},
			{"node a2", 192, []float64{76.2031}},
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
		closed := slices.ContainsFunc(p.Segs, func(s scene.Seg) bool { return s.Op == scene.Close })
		switch {
		case p.Fill != (scene.Paint{}) && !closed: // the line's own fill
		case p.Fill != (scene.Paint{}) && curved:
			b := line.Bounds(p.Segs)
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
		case len(p.Stroke.Dash) > 0 && p.Stroke.Dash[0] != 0: // the author's: see wantStyle
			g.width = p.Stroke.Width
		case len(p.Stroke.Dash) > 0:
			g.width = p.Stroke.Width
			for l := 0.0; l <= line.ArcLength(p.Segs)+1e-9; l += p.Stroke.Dash[0] + p.Stroke.Dash[1] {
				dots = append(dots, along(p.Segs, l))
			}
		default:
			g.width = p.Stroke.Width
			l := line.ArcLength(p.Segs)
			for _, at := range []float64{0, min(2, l), max(l-2, 0), l} {
				q := along(p.Segs, at)
				g.solids = append(g.solids, q.X, q.Y)
			}
			for i := 1; i+1 < len(p.Segs); i++ { // a hop is two curves after a line
				if p.Segs[i].Op == scene.CubeTo && p.Segs[i+1].Op == scene.CubeTo {
					from, top, end := p.Segs[i-1].Pts[0], p.Segs[i].Pts[2], p.Segs[i+1].Pts[2]
					q := line.Bend{from, p.Segs[i].Pts[0], p.Segs[i].Pts[1], top}.At(0.5)
					g.hops = append(g.hops, from.X, from.Y, top.X, top.Y, end.X, end.Y, q.X, q.Y)
				}
			}
		}
	}
	if n := len(dots); n > 0 {
		g.dots = []float64{float64(n), dots[0].X, dots[0].Y, dots[n-1].X, dots[n-1].Y}
	}
	return g
}

func colorOf(p scene.Paint) string {
	if p.Token != scene.None {
		return ""
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", p.Color.R, p.Color.G, p.Color.B, p.Color.A)
}

// styleOf is what an item paints, as wantStyle has it. A line's paths are its fill
// when it has one, its strokes, then its markers.
func styleOf(it scene.Item) wantStyle {
	g := wantStyle{fill: colorOf(scene.Paint{}), stroke: colorOf(scene.Paint{})}
	if len(it.Texts) > 0 {
		g.text = colorOf(it.Texts[0].Paint)
	}
	var dashes [][4]float64
	for i, p := range it.Paths {
		closed := slices.ContainsFunc(p.Segs, func(s scene.Seg) bool { return s.Op == scene.Close })
		moves := 0
		for _, s := range p.Segs {
			if s.Op == scene.MoveTo {
				moves++
			}
		}
		switch {
		case it.Kind == scene.Edge && p.Fill != (scene.Paint{}) && closed:
			g.marks = append(g.marks, colorOf(p.Fill))
			continue
		case it.Kind == scene.Edge && moves > 1:
			g.marks = append(g.marks, colorOf(p.Stroke.Paint))
			continue
		case i == 0 && p.Fill != (scene.Paint{}):
			g.fill = colorOf(p.Fill)
		}
		if p.Stroke.Width > 0 && p.Stroke.Paint != (scene.Paint{}) && g.width == 0 {
			g.stroke, g.width = colorOf(p.Stroke.Paint), p.Stroke.Width
		}
		if p.Stroke.Width > 0 && len(p.Stroke.Dash) == 0 && len(p.Segs) > 0 && line.ArcLength(p.Segs) == 0 { // a dot
			q := p.Segs[0].Pts[0]
			dashes = append(dashes, [4]float64{q.X, q.Y, q.X, q.Y})
		}
		if d := p.Stroke.Dash; len(d) > 0 && p.Stroke.Paint != (scene.Paint{}) {
			l := line.ArcLength(p.Segs)
			for a, k := 0.0, 0; a <= l+1e-9; k++ {
				if k%2 == 0 {
					q, r := along(p.Segs, a), along(p.Segs, min(a+d[k%len(d)], l))
					dashes = append(dashes, [4]float64{q.X, q.Y, r.X, r.Y})
				}
				a += d[k%len(d)]
			}
		}
	}
	if n := len(dashes); n > 0 {
		first, last := dashes[0], dashes[n-1]
		g.dashes = append(append([]float64{float64(n)}, first[:]...), last[:]...)
	}
	return g
}

// sameStyle says how an item's paint differs from w's, or "". A curve's dashes may be
// up to 1 off: Chrome measures a curve as coarse chords, and sumi its length.
func sameStyle(it scene.Item, w wantStyle) string {
	g := styleOf(it)
	if w.text == "on" {
		c := it.Paths[0].Fill.Color
		w.text = colorOf(scene.Fixed(scene.TextOn(c)))
	}
	tol := 0.01
	if len(it.Paths) > 0 && slices.ContainsFunc(it.Paths[0].Segs, func(s scene.Seg) bool { return s.Op == scene.CubeTo }) {
		tol = 1
	}
	dashes := w.dashes == nil || near(g.dashes, w.dashes, tol)
	switch {
	case it.Kind == scene.Label:
		if g.text != w.text {
			return fmt.Sprintf("text %s", g.text)
		}
	case g.fill != w.fill || g.stroke != w.stroke || g.text != w.text || math.Abs(g.width-w.width) > 0.01 || !slices.Equal(g.marks, w.marks) || !dashes:
		return fmt.Sprintf("%+v", g)
	}
	return ""
}

// along is the point at length l along segs.
func along(segs []scene.Seg, l float64) scene.Point {
	if rest := line.Span(segs, l, math.Inf(1)); len(rest) > 0 {
		return rest[0].Pts[0]
	}
	bs := line.Bends(segs)
	return bs[len(bs)-1].End()
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
		a, _ := label.Extent(scene.Arial, t.Font)
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
			if !near(g.box[:], w.box[:], 0.01) || math.Abs(g.width-w.width) > 0.01 || !near(g.solids, w.solids, 0.01) || !near(g.hops, w.hops, 0.1) || !near(g.dots, w.dots, 0.01) ||
				!holds(g.marks, w.marks) || !near(g.rings, w.rings, 0.01) || math.Abs(g.cross-w.cross) > 0.01 {
				t.Errorf("%s: edge %s draws\n %+v\nwant %+v", c.name, id, g, w)
			}
		}
		for key, w := range c.styles {
			if m := sameStyle(items[key], w); m != "" {
				t.Errorf("%s: %s paints %s\nwant %+v", c.name, key, m, w)
			}
		}
		for id, w := range c.groups {
			if b := line.Bounds(items["group "+id].Paths[0].Segs); !near([]float64{b.X, b.Y, b.W, b.H}, w[:], 0.01) {
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
