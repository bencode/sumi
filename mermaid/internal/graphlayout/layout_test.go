package graphlayout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/elk"
)

// Every graph here is one Mermaid laid out, and what is expected is what Mermaid gave
// its painter for it.

func sized(id, parent string, width, height float64) Node {
	return Node{ID: id, Parent: parent, Width: width, Height: height}
}

func group(n Node) Node {
	n.Group = true
	return n
}

func marked(e Edge, start, end string) Edge {
	e.StartMarker, e.EndMarker = start, end
	return e
}

func describeLayout(r Result) string {
	lines := make([]string, 0, len(r.Nodes)+len(r.Edges))
	for _, n := range r.Nodes {
		lines = append(lines, fmt.Sprintf("%s %v,%v %vx%v", n.ID, n.X, n.Y, n.Width, n.Height))
	}
	for _, e := range r.Edges {
		var b strings.Builder
		b.WriteString(e.ID + ":")
		for _, p := range e.Points {
			fmt.Fprintf(&b, " %v,%v", p.X, p.Y)
		}
		if e.Label != nil {
			fmt.Fprintf(&b, " label %v,%v", e.Label.X, e.Label.Y)
		}
		lines = append(lines, b.String())
	}
	return strings.Join(lines, "\n")
}

func TestLayout(t *testing.T) {
	tests := []struct {
		name string
		data Data
		want string
	}{
		{
			"edges across subgraphs and their frames pulled in",
			Data{Direction: elk.Left, Nodes: []Node{
				group(subgraph("s0", "", elk.Undefined, &elk.Size{Width: 49, Height: 21})),
				group(subgraph("s2", "s0", elk.Undefined, &elk.Size{Width: 307.40625, Height: 21})),
				group(subgraph("s1", "s0", elk.Right, &elk.Size{Width: 307.40625, Height: 21})),
				group(subgraph("s3", "s1", elk.Undefined, &elk.Size{Width: 31.125, Height: 21})),
				sized("n0", "s3", 152, 45),
				sized("n4", "s2", 152, 45),
				sized("n2", "s0", 152, 45),
				sized("n1", "", 152, 45),
				sized("n3", "", 152, 45),
			}, Edges: []Edge{
				marked(edge("L_n1_n4_0", "n1", "n4"), "none", "arrow_point"),
				marked(edge("L_n4_n1_0", "n4", "n1"), "none", "arrow_point"),
				marked(edge("L_n2_n4_0", "n2", "n4"), "none", "arrow_point"),
				marked(edge("L_n4_n2_0", "n4", "n2"), "none", "arrow_point"),
				marked(edge("L_n4_n2_2", "n4", "n2"), "none", "arrow_point"),
				marked(edge("L_n0_n3_0", "n0", "n3"), "none", "arrow_point"),
				marked(edge("L_n3_n0_0", "n3", "n0"), "none", "arrow_point"),
				marked(labelled("L_s0_s2_0", "s0", "s2", 42.828125, 21), "none", "arrow_point"),
				marked(labelled("L_n0_s2_0", "n0", "s2", 42.828125, 21), "none", "arrow_point"),
				marked(labelled("L_s2_n0_0", "s2", "n0", 42.828125, 21), "none", "arrow_point"),
				marked(edge("L_n0_s1_0", "n0", "s1"), "none", "arrow_point"),
			}},
			`s0 867.1171875,150 1178.234375x276
s2 1027.53125,163.35 317.40625x144
s1 537,162 318x204
s3 502,167.25 200x118.5
n0 502,180 152x45
n4 1014.234375,175.35 152x45
n2 1356.234375,155.02499999999998 152x45
n1 1582.234375,200.02499999999998 152x45
n3 88,174 152x45
L_n1_n4_0: 1506.234375,195.85833333333332 1466.234375,195.85833333333332 1250.234375,195.85833333333332 1250.234375,187.35 1196.234375,187.35 1132.234375,187.35 1132.234375,179.25 1090.234375,179.25
L_n4_n1_0: 1090.234375,183.14999999999998 1120.234375,183.14999999999998 1120.234375,199.35 1196.234375,199.35 1250.234375,199.35 1250.234375,204.19166666666666 1466.234375,204.19166666666666 1506.234375,204.19166666666666
L_n2_n4_0: 1280.234375,149.175 1226.234375,149.175 1226.234375,151.35 1196.234375,151.35 1120.234375,151.35 1120.234375,167.55 1090.234375,167.55
L_n4_n2_0: 1090.234375,171.45 1132.234375,171.45 1132.234375,163.35 1196.234375,163.35 1238.234375,163.35 1238.234375,155.02499999999998 1280.234375,155.02499999999998
L_n4_n2_2: 1090.234375,175.35 1196.234375,175.35 1250.234375,175.35 1250.234375,160.875 1280.234375,160.875
L_n0_n3_0: 426,174.15 332,174.15 238,174.15 204,174.15 184,174.15 184,169.83333333333334 164,169.83333333333334
L_n3_n0_0: 164,178.16666666666669 184,178.16666666666669 184,180 204,180 238,180 332,180 426,180
L_s0_s2_0: 1456.234375,234.52499999999998 1238.234375,234.52499999999998 1238.234375,211.35 1186.234375,211.35 label 1356.234375,234.02499999999998
L_n0_s2_0: 578,176.1 672,176.1 706,176.1 736,176.1 736,174 868.828125,174 label 764,174
L_s2_n0_0: 868.828125,186 736,186 736,183.9 706,183.9 672,183.9 578,183.9 label 810.828125,186
L_n0_s1_0: 426,185.85000000000002 332,185.85000000000002 302,185.85000000000002 302,240 696,240`,
		},
		{
			"empty subgraphs and self loops",
			Data{Direction: elk.Left, Nodes: []Node{
				group(subgraph("s1", "", elk.Undefined, &elk.Size{Width: 307.40625, Height: 21})),
				group(subgraph("s0", "", elk.Undefined, &elk.Size{Width: 307.40625, Height: 21})),
				group(subgraph("s2", "s0", elk.Undefined, &elk.Size{Width: 307.40625, Height: 21})),
				group(subgraph("s3", "s2", elk.Undefined, &elk.Size{Width: 3.890625, Height: 21})),
				sized("n2", "s2", 152, 45),
				sized("n0", "s0", 152, 45),
				sized("n3", "s0", 152, 45),
				sized("n4", "s0", 152, 45),
				sized("n1", "", 152, 45),
				sized("n5", "", 152, 45),
			}, Edges: []Edge{
				marked(edge("L_n0_n0_0", "n0", "n0"), "none", "arrow_point"),
				marked(edge("L_n3_n0_0", "n3", "n0"), "none", "arrow_point"),
				marked(edge("L_n3_n3_0", "n3", "n3"), "none", "arrow_point"),
				marked(edge("L_s3_n3_0", "s3", "n3"), "none", "arrow_point"),
				marked(edge("L_n3_s3_0", "n3", "s3"), "none", "arrow_point"),
				marked(labelled("L_s3_s0_0", "s3", "s0", 42.828125, 21), "none", "arrow_point"),
				marked(labelled("L_n4_n2_0", "n4", "n2", 42.828125, 21), "none", "arrow_point"),
				marked(edge("L_s3_s0_2", "s3", "s0"), "none", "arrow_point"),
				marked(edge("L_n2_s1_0", "n2", "s1"), "none", "arrow_point"),
				marked(edge("L_n0_n5_0", "n0", "n5"), "none", "arrow_point"),
			}},
			`s1 164,215.5 307.40625x0
s0 728.1171875,249.78 1028.234375x305.56
s2 807.703125,203.5 315.40625x117
s3 789.40625,300.96000000000004 3.890625x0
n2 865.40625,215.5 152x45
n0 314,279.65999999999997 152x45
n3 490,279.65999999999997 152x45
n4 1142.234375,215.5 152x45
n1 723.1171875,34.5 152x45
n5 88,279.65999999999997 152x45
L_n0_n0_0: 335.3333333333333,257.15999999999997 335.33333333333337,245.16 292.66666666666663,245.16 292.66666666666663,257.15999999999997
L_n3_n0_0: 414,279.65999999999997 390,279.65999999999997
L_n3_n3_0: 511.3333333333333,257.15999999999997 511.33333333333337,245.16 468.66666666666663,245.16 468.66666666666663,257.15999999999997
L_s3_n3_0: 789.40625,300.96000000000004 759.40625,300.96000000000004 759.40625,271.56 638,271.56 608,271.56 608,275.76 566,275.76
L_n3_s3_0: 566,283.56 638,283.56 735.40625,283.56 735.40625,296.64 789.40625,296.64
L_s3_s0_0: 789.40625,288 747.40625,288 747.40625,307.56 638,307.56 608,307.56 608,355.56 1242.234375,355.56 label 1020.8203125,355.06
L_n4_n2_0: 1066.234375,215.5 975.40625,215.5 941.40625,215.5 label 1020.8203125,215
L_s3_s0_2: 789.40625,292.32 723.40625,292.32 723.40625,295.56 638,295.56 596,295.56 596,378.56 1242.234375,378.56
L_n2_s1_0: 789.40625,215.5 638,215.5 204,215.5 164,215.5
L_n0_n5_0: 238,279.65999999999997 204,279.65999999999997 164,279.65999999999997`,
		},
		{
			"a title wider than its frame",
			Data{Direction: elk.Up, Nodes: []Node{
				group(subgraph("s1", "", elk.Right, &elk.Size{Width: 385, Height: 21})),
				group(subgraph("s0", "", elk.Undefined, &elk.Size{Width: 266, Height: 21})),
				sized("n0_0", "s0", 152, 45),
				sized("n1_0", "s1", 152, 45),
				sized("o0", "", 152, 45),
				sized("o2", "", 152, 45),
				sized("o1", "", 152, 45),
			}, Edges: []Edge{
				marked(labelled("L_o0_o0_0", "o0", "o0", 15.578125, 21), "arrow_circle", "arrow_circle"),
				labelled("L_o2_o1_0", "o2", "o1", 15.578125, 21),
				edge("L_s0_s0_0", "s0", "s0"),
			}},
			`s1 208.5,296 393x117
s0 600,296 200x276
n0_0 600,387.5 152x45
n1_0 112,308 152x45
o0 888.578125,296 152x45
o2 1080.578125,180.5 152x45
o1 1080.578125,34.5 152x45
L_o0_o0_0: 812.578125,299.5 792.578125,299.5 792.578125,292.5 812.578125,292.5 label 780.7890625,296
L_o2_o1_0: 1080.578125,158 1080.578125,57 label 1080.3671875,107.5
L_s0_s0_0: 463,338 447,338 447,254 463,254`,
		},
		{
			"a label keeps an edge from straightening",
			Data{Direction: elk.Right, Nodes: []Node{
				group(subgraph("s0", "", elk.Right, &elk.Size{Width: 3.890625, Height: 21})),
				sized("n0", "s0", 152, 45),
				sized("n2", "s0", 152, 45),
				sized("n1", "", 152, 45),
			}, Edges: []Edge{
				labelled("L_n2_n0_0", "n2", "n0", 25.6875, 21),
				edge("L_n0_n2_0", "n0", "n2"),
			}},
			`s0 260.84375,70.5 497.6875x117
n0 112,82.5 152x45
n2 409.6875,82.5 152x45
n1 260.84375,192 152x45
L_n2_n0_0: 333.6875,86.4 218,86.4 218,86.4 188,86.4 label 260.84375,86.4
L_n0_n2_0: 188,78.6 218,78.6 218,70.5 303.6875,70.5 303.6875,78.6 333.6875,78.6`,
		},
		{
			"a frame pulled in at the bottom",
			Data{Direction: elk.Down, Nodes: []Node{
				group(subgraph("s1", "", elk.Undefined, &elk.Size{Width: 3.890625, Height: 21})),
				group(subgraph("s0", "", elk.Undefined, &elk.Size{Width: 378, Height: 21})),
				sized("n0_0", "s0", 152, 45),
				sized("n1_0", "s1", 152, 45),
				sized("o2", "", 152, 45),
			}, Edges: []Edge{
				marked(edge("L_s1_s0_0", "s1", "s0"), "none", "arrow_point"),
				edge("L_n1_0_o2_0", "n1_0", "o2"),
				marked(labelled("L_o2_n1_0_0", "o2", "n1_0", 15.578125, 21), "arrow_cross", "arrow_cross"),
			}},
			`s1 426.73333333333335,70.5 200x117
s0 201,257.5 200x117
n0_0 201,269.5 152x45
n1_0 426.73333333333335,82.5 152x45
o2 431.73333333333335,649.5 152x45
L_s1_s0_0: 460.46666666666664,129 460.46666666666664,159 201,159 201,199
L_n1_0_o2_0: 405,105 405,139 405,179 409.73333333333335,179 409.73333333333335,393 409.73333333333335,627
L_o2_n1_0_0: 453.73333333333335,627 453.73333333333335,179 448.46666666666664,179 448.46666666666664,139 448.46666666666664,105 label 453.52239583333335,393`,
		},
		{
			"stubs dropped and opposite labels at different heights",
			Data{Direction: elk.Right, Nodes: []Node{
				group(subgraph("s1", "", elk.Undefined, &elk.Size{Width: 307.40625, Height: 21})),
				group(subgraph("s0", "", elk.Undefined, &elk.Size{Width: 31.125, Height: 21})),
				group(subgraph("s2", "s0", elk.Down, &elk.Size{Width: 147, Height: 21})),
				sized("n1", "s2", 152, 45),
				sized("n6", "s2", 152, 45),
				sized("n9", "s2", 152, 45),
				sized("n0", "s0", 152, 45),
				sized("n2", "s0", 152, 45),
				sized("n3", "s0", 152, 45),
				sized("n5", "s0", 152, 45),
				sized("n4", "s1", 152, 45),
				sized("n7", "", 152, 45),
				sized("n8", "", 152, 45),
			}, Edges: []Edge{
				marked(edge("L_n0_n5_0", "n0", "n5"), "none", "arrow_point"),
				marked(labelled("L_n3_n0_0", "n3", "n0", 25.6875, 21), "none", "arrow_point"),
				marked(labelled("L_n2_n5_0", "n2", "n5", 25.6875, 21), "none", "arrow_point"),
				edge("L_n5_n5_0", "n5", "n5"),
				edge("L_n5_n5_2", "n5", "n5"),
				marked(edge("L_n6_n2_0", "n6", "n2"), "none", "arrow_point"),
				marked(edge("L_s0_n7_0", "s0", "n7"), "none", "arrow_point"),
				marked(edge("L_n5_n1_0", "n5", "n1"), "none", "arrow_point"),
				marked(edge("L_n1_n5_0", "n1", "n5"), "none", "arrow_point"),
				marked(labelled("L_n3_n0_2", "n3", "n0", 42.828125, 21), "none", "arrow_point"),
				marked(edge("L_s1_n5_0", "s1", "n5"), "none", "arrow_point"),
			}},
			`s1 170.703125,194.42499999999998 317.40625x117
s0 1024.8203125,232 1290.828125x440
s2 1324.234375,213.5 200x307
n1 1324.234375,130.5 152x45
n6 1324.234375,320.5 152x45
n9 1324.234375,225.5 152x45
n0 794.234375,133.325 152x45
n2 1570.234375,374.25 152x45
n3 479.40625,133.325 152x45
n5 1006.234375,137.22499999999997 152x45
n4 112,206.42499999999998 152x45
n7 1786.234375,213.825 152x45
n8 170.703125,315.42499999999995 152x45
L_n0_n5_0: 870.234375,133.325 930.234375,133.325
L_n3_n0_0: 555.40625,137.22499999999997 585.40625,137.22499999999997 585.40625,171.42499999999998 688.234375,171.42499999999998 688.234375,137.22499999999997 718.234375,137.22499999999997 label 636.8203125,170.92499999999998
L_n2_n5_0: 1494.234375,378.15 1464.234375,378.15 1464.234375,428 1112.234375,428 1112.234375,143.075 1082.234375,143.075 label 1294.234375,427.5
L_n5_n5_0: 997.8865489130435,114.72499999999997 993.434375,114.72499999999998 993.434375,102.72499999999998 1019.034375,102.72499999999998 1019.034375,114.72499999999998 1014.5822010869565,114.72499999999997
L_n5_n5_2: 990.3811639908257,114.72499999999997 967.834375,114.72499999999998 967.834375,82.72499999999998 1044.634375,82.72499999999998 1044.634375,114.72499999999998 1022.0875860091744,114.72499999999997
L_n6_n2_0: 1400.234375,320.5 1434.234375,320.5 1464.234375,320.5 1464.234375,370.35 1494.234375,370.35
L_s0_n7_0: 1670.234375,213.825 1710.234375,213.825
L_n5_n1_0: 1082.234375,137.22499999999997 1124.234375,137.22499999999997 1124.234375,134.4 1154.234375,134.4 1248.234375,134.4
L_n1_n5_0: 1248.234375,126.6 1154.234375,126.6 1112.234375,126.6 1112.234375,131.375 1082.234375,131.375
L_n3_n0_2: 555.40625,129.42499999999998 718.234375,129.42499999999998 label 636.8203125,128.92499999999998
L_s1_n5_0: 329.40625,194.42499999999998 369.40625,194.42499999999998 900.234375,194.42499999999998 900.234375,141.125 930.234375,141.125`,
		},
		{
			"a frame held open by its title",
			Data{Direction: elk.Down, Nodes: []Node{
				group(subgraph("s0", "", elk.Undefined, &elk.Size{Width: 307.40625, Height: 21})),
				group(subgraph("s1", "s0", elk.Left, &elk.Size{Width: 307.40625, Height: 21})),
				sized("n2", "s1", 152, 45),
				sized("n0", "s0", 152, 45),
				sized("n1", "s0", 152, 45),
			}, Edges: []Edge{
				edge("L_n1_n0_0", "n1", "n0"),
				marked(edge("L_n0_n1_0", "n0", "n1"), "none", "arrow_point"),
				marked(edge("L_n2_n0_0", "n2", "n0"), "none", "arrow_point"),
				marked(edge("L_n2_s0_0", "n2", "s0"), "none", "arrow_point"),
			}},
			`s0 180.8515625,298.703125 315.40625x573.40625
s1 213.703125,118.5 200x117
n2 213.703125,130.5 152x45
n0 235.43645833333332,469.90625 152x45
n1 235.43645833333332,538.90625 152x45
L_n1_n0_0: 257.16979166666664,516.40625 257.16979166666664,492.40625
L_n0_n1_0: 213.70312499999997,492.40625 213.70312499999997,516.40625
L_n2_n0_0: 235.4364583333333,153 235.4364583333333,387.40625 235.4364583333333,447.40625
L_n2_s0_0: 191.96979166666665,153 191.96979166666665,387.40625 191.96979166666665,417.40625 48,417.40625 48,12`,
		},
		{
			"small nodes moved onto their lines",
			Data{Direction: elk.Down, Nodes: []Node{
				sized("root_start", "", 14, 14),
				sized("S0", "", 136, 37),
				sized("S1", "", 136, 37),
				sized("S2", "", 136, 37),
				sized("root_end", "", 14.013293266296387, 14),
			}, Edges: []Edge{
				marked(edge("edge0", "root_start", "S0"), "none", "arrow_barb_neo"),
				marked(edge("edge1", "S0", "S1"), "none", "arrow_barb_neo"),
				marked(labelled("edge2", "S0", "S2", 22.578125, 21), "none", "arrow_barb_neo"),
				marked(edge("edge3", "S0", "root_end"), "none", "arrow_barb_neo"),
			}},
			`root_start 199.5,19 14x14
S0 199.5,84.5 136x37
S1 80,161.5 136x37
S2 199.5,238.5 136x37
root_end 260.578125,150 14.013293266296387x14
edge0: 199.5,26 199.5,66
edge1: 170.5,103 170.5,123 80,123 80,143
edge2: 199.5,103 199.5,220 label 199.2890625,161.5
edge3: 228.5,103 228.5,123 260.578125,123 260.578125,143`,
		},
		{
			"an edge ending on a frame",
			Data{Direction: elk.Down, Nodes: []Node{
				group(subgraph("s1", "", elk.Right, &elk.Size{Width: 307.40625, Height: 21})),
				group(subgraph("s0", "", elk.Right, &elk.Size{Width: 307.40625, Height: 21})),
				sized("n0_0", "s0", 152, 45),
				sized("n1_0", "s1", 152, 45),
				sized("n1_1", "s1", 152, 45),
				sized("o0", "", 152, 45),
			}, Edges: []Edge{
				edge("L_s1_o0_0", "s1", "o0"),
				marked(edge("L_s0_s1_0", "s0", "s1"), "arrow_cross", "arrow_cross"),
				marked(labelled("L_n0_0_n0_0_0", "n0_0", "n0_0", 15.578125, 21), "none", "arrow_point"),
			}},
			`s1 169.703125,297.4 315.40625x185.99999999999997
s0 169.703125,88.2 315.40625x152.4
n0_0 112,117.9 152x45
n1_0 112,274.9 152x45
n1_1 112,343.9 152x45
o0 170.703125,452.9 152x45
L_s1_o0_0: 170.703125,390.4 170.703125,430.4
L_s0_s1_0: 170.703125,164.4 170.703125,204.4
L_n0_0_n0_0_0: 90.66666666666667,95.4 90.66666666666667,83.4 133.33333333333334,83.4 133.33333333333334,95.4 label 112,70.5`,
		},
		{
			"a crossing keeps an edge from straightening",
			Data{Direction: elk.Left, Nodes: []Node{
				group(subgraph("s2", "", elk.Right, &elk.Size{Width: 3.890625, Height: 21})),
				group(subgraph("s1", "", elk.Undefined, &elk.Size{Width: 343, Height: 21})),
				group(subgraph("s0", "", elk.Right, &elk.Size{Width: 307.40625, Height: 21})),
				sized("n0_0", "s0", 152, 45),
				sized("n1_0", "s1", 152, 45),
				sized("n1_1", "s1", 152, 45),
				sized("n2_0", "s2", 152, 45),
				sized("o1", "", 152, 45),
				sized("o0", "", 152, 45),
				sized("o2", "", 152, 45),
			}, Edges: []Edge{
				marked(labelled("L_o1_s1_0", "o1", "s1", 15.578125, 21), "arrow_cross", "arrow_cross"),
				marked(edge("L_o1_n1_0_0", "o1", "n1_0"), "none", "arrow_point"),
				edge("L_o0_s2_0", "o0", "s2"),
				marked(edge("L_o0_n1_0_0", "o0", "n1_0"), "arrow_circle", "arrow_circle"),
				marked(edge("L_o2_n1_0_0", "o2", "n1_0"), "arrow_point", "arrow_point"),
			}},
			`s2 265,70.5 200x117
s1 189.5,275.375 351x212.75
s0 804.703125,466.75 315.40625x117
n0_0 747,478.75 152x45
n1_0 193,334.5 152x45
n1_1 193,239.5 152x45
n2_0 265,82.5 152x45
o1 723,345.75 152x45
o0 531,74.66666666666667 152x45
o2 531,330.75 152x45
L_o1_s1_0: 647,349.91666666666663 627,349.91666666666663 627,404.25 435,404.25 435,357.75 365,357.75 label 531,403.75
L_o1_n1_0_0: 647,341.58333333333326 627,341.58333333333326 627,287.25 531,287.25 415,287.25 415,333.75 375,333.75 311,333.75 311,334.5 269,334.5
L_o0_s2_0: 455,70.5 365,70.5
L_o0_n1_0_0: 455,78.83333333333334 395,78.83333333333334 395,328.65 375,328.65 269,328.65
L_o2_n1_0_0: 455,330.75 435,330.75 435,345.75 375,345.75 311,345.75 311,340.35 269,340.35`,
		},
		{
			"a stub dropped at the start of an edge",
			Data{Direction: elk.Up, Nodes: []Node{
				group(subgraph("s0", "", elk.Down, &elk.Size{Width: 3.890625, Height: 21})),
				sized("n0_0", "s0", 152, 45),
				sized("n0_1", "s0", 152, 45),
				sized("o2", "", 152, 45),
				sized("o0", "", 152, 45),
				sized("o1", "", 152, 45),
			}, Edges: []Edge{
				marked(edge("L_n0_0_s0_0", "n0_0", "s0"), "arrow_circle", "arrow_circle"),
				marked(edge("L_o2_o0_0", "o2", "o0"), "arrow_circle", "arrow_circle"),
				marked(edge("L_n0_0_n0_0_0", "n0_0", "n0_0"), "arrow_cross", "arrow_cross"),
				marked(edge("L_o0_o1_0", "o0", "o1"), "arrow_point", "arrow_point"),
				marked(edge("L_n0_0_n0_1_0", "n0_0", "n0_1"), "arrow_point", "arrow_point"),
				edge("L_s0_n0_0_0", "s0", "n0_0"),
				marked(edge("L_o1_o1_0", "o1", "o1"), "arrow_cross", "arrow_cross"),
				marked(edge("L_o2_o0_2", "o2", "o0"), "arrow_cross", "arrow_cross"),
				marked(edge("L_o0_o1_2", "o0", "o1"), "arrow_circle", "arrow_circle"),
			}},
			`s0 134,293 244x222
n0_0 156,357.5 152x45
n0_1 112,252.5 152x45
o2 372,204.5 152x45
o0 372,119.5 152x45
o1 372,34.5 152x45
L_n0_0_s0_0: 177.7333333333333,335 177.73333333333332,305 200,305 200,182
L_o2_o0_0: 350,182 350,142
L_n0_0_n0_0_0: 80,360.52272727272725 68.00000000000001,361 68.00000000000001,354 80.00000000000001,354
L_o0_o1_0: 350,97 350,57
L_n0_0_n0_1_0: 134.26666666666668,335 134.26666666666668,305 111.99999999999999,305 111.99999999999999,275
L_s0_n0_0_0: 156,404 156,380
L_o1_o1_0: 296,38 276,38 276,31 296,31
L_o2_o0_2: 394,182 394,142
L_o0_o1_2: 394,97 394,57`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := describeLayout(Layout(test.data)); got != test.want {
				t.Errorf("got\n%s\nwant\n%s", got, test.want)
			}
		})
	}
}
