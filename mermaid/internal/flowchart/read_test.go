package flowchart

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// readCases read as they do in Mermaid v12.3.0: each want is what its getData gave.
var readCases = []struct {
	name, src string
	want      []string
}{
	{"every shape", "flowchart TD\nA[a]\nB(b)\nC([c])\nD[[d]]\nE[(e)]\nF((f))\nG(((g)))\nH{h}\nI{{i}}\nJ>j]\nK[/k\\]\nL[\\l/]\nM[/m/]\nN[\\n\\]\nO[|a:b|o]", []string{
		"TB",
		"node A rect \"a\"/text",
		"node B round \"b\"/text",
		"node C stadium \"c\"/text",
		"node D subroutine \"d\"/text",
		"node E cylinder \"e\"/text",
		"node F circle \"f\"/text",
		"node G doublecircle \"g\"/text",
		"node H diamond \"h\"/text",
		"node I hexagon \"i\"/text",
		"node J odd \"j\"/text",
		"node K trapezoid \"k\"/text",
		"node L invtrapezoid \"l\"/text",
		"node M leanright \"m\"/text",
		"node N leanleft \"n\"/text",
		"node O rect \"o\"/text",
	}},
	{"a circle or a cross on a line goes with the line", "flowchart TD\nA---oB\nC --- oD\nE---xF", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"node C rect \"C\"/text",
		"node oD rect \"oD\"/text",
		"node E rect \"E\"/text",
		"node F rect \"F\"/text",
		"edge L_A_B_0 A B normal none arrow_circle \"\"/text",
		"edge L_C_oD_0 C oD normal none none \"\"/text",
		"edge L_E_F_0 E F normal none arrow_cross \"\"/text",
	}},
	{"markers and strokes", "flowchart TD\nA-->B\nB---C\nC-.->D\nD==>E\nE~~~F\nF--xG\nG--oH", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"node C rect \"C\"/text",
		"node D rect \"D\"/text",
		"node E rect \"E\"/text",
		"node F rect \"F\"/text",
		"node G rect \"G\"/text",
		"node H rect \"H\"/text",
		"edge L_A_B_0 A B normal none arrow_point \"\"/text",
		"edge L_B_C_0 B C normal none none \"\"/text",
		"edge L_C_D_0 C D dotted none arrow_point \"\"/text",
		"edge L_D_E_0 D E thick none arrow_point \"\"/text",
		"edge L_E_F_0 E F invisible none none \"\"/text",
		"edge L_F_G_0 F G normal none arrow_cross \"\"/text",
		"edge L_G_H_0 G H normal none arrow_circle \"\"/text",
	}},
	{"both ends", "flowchart TD\nA<-->B\nB o--o C\nC x--x D\nD <-. t .-> E\nE x-- t --x F\nF o== t ==o G", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"node C rect \"C\"/text",
		"node D rect \"D\"/text",
		"node E rect \"E\"/text",
		"node F rect \"F\"/text",
		"node G rect \"G\"/text",
		"edge L_A_B_0 A B normal arrow_point arrow_point \"\"/text",
		"edge L_B_C_0 B C normal arrow_circle arrow_circle \"\"/text",
		"edge L_C_D_0 C D normal arrow_cross arrow_cross \"\"/text",
		"edge L_D_E_0 D E dotted arrow_point arrow_point \"t\"/text",
		"edge L_E_F_0 E F normal arrow_cross arrow_cross \"t\"/text",
		"edge L_F_G_0 F G thick arrow_circle arrow_circle \"t\"/text",
	}},
	{"ends that disagree", "flowchart TD\nA x-- t --> B\nC == t x==> D", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"node C rect \"C\"/text",
		"node D rect \"D\"/text",
		"edge L_A_B_0 A B INVALID none arrow_point \"t\"/text",
		"edge L_C_D_0 C D INVALID none arrow_point \"t\"/text",
	}},
	{"words that are almost keywords", "flowchart TD\nv --> o\nend1 --> default", []string{
		"TB",
		"node v rect \"v\"/text",
		"node o rect \"o\"/text",
		"node end1 rect \"end1\"/text",
		"node default rect \"default\"/text",
		"edge L_v_o_0 v o normal none arrow_point \"\"/text",
		"edge L_end1_default_0 end1 default normal none arrow_point \"\"/text",
	}},
	{"the strings of a label", "flowchart TD\nA[\"a\"b]\nB[\"`md`\"]\nC-- \"`md`\" -->D\nE-->|\"x\"|F", []string{
		"TB",
		"node A rect \"ab\"/string",
		"node B rect \"md\"/markdown",
		"node C rect \"C\"/text",
		"node D rect \"D\"/text",
		"node E rect \"E\"/text",
		"node F rect \"F\"/text",
		"edge L_C_D_0 C D normal none arrow_point \"md\"/markdown",
		"edge L_E_F_0 E F normal none arrow_point \"x\"/string",
	}},
	{"a line that mentions a direction is a direction", "flowchart TD\nA --> B direction LR\nA[direction LR]", []string{
		"TB",
	}},
	{"later text wins, plain mentions keep it", "flowchart TD\nA[x]\nA\nA[y]\nA(z)", []string{
		"TB",
		"node A round \"z\"/text",
	}},
	{"style makes a vertex", "flowchart TD\nstyle Z fill:#f00\nZ[x]", []string{
		"TB",
		"node Z rect \"x\"/text styles=fill:#f00",
	}},
	{"class does not", "flowchart TD\nclass Z c\nZ", []string{
		"TB",
		"node Z rect \"Z\"/text",
	}},
	{"the ; after a color goes", "flowchart TD\nstyle A fill:#f00;\nB", []string{
		"TB",
		"node A rect \"A\"/text styles=fill:#f00",
		"node B rect \"B\"/text",
	}},
	{"an id goes to the last start and first end", "flowchart TD\nA  & B e1@--> C & D", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"node C rect \"C\"/text",
		"node D rect \"D\"/text",
		"edge L_A_C_0 A C normal none arrow_point \"\"/text",
		"edge L_A_D_0 A D normal none arrow_point \"\"/text",
		"edge e1 B C normal none arrow_point \"\"/text",
		"edge L_B_D_0 B D normal none arrow_point \"\"/text",
	}},
	{"a repeated start is its last too", "flowchart TD\nA & A e1@--> B", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"edge e1 A B normal none arrow_point \"\"/text",
		"edge L_A_B_2 A B normal none arrow_point \"\"/text",
	}},
	{"edge numbers skip 1", "flowchart TD\nA --> B\nA --> B\nA --> B", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"edge L_A_B_0 A B normal none arrow_point \"\"/text",
		"edge L_A_B_2 A B normal none arrow_point \"\"/text",
		"edge L_A_B_3 A B normal none arrow_point \"\"/text",
	}},
	{"a taken id is not used", "flowchart TD\nA e1@--> B\nA e1@--> B", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"edge e1 A B normal none arrow_point \"\"/text",
		"edge L_A_B_2 A B normal none arrow_point \"\"/text",
	}},
	{"an edge id is no vertex", "flowchart TD\nA e1@--> B\ne1 --> C", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"node C rect \"C\"/text",
		"edge e1 A B normal none arrow_point \"\"/text",
		"edge L_e1_C_0 e1 C normal none arrow_point \"\"/text",
	}},
	{"style on an edge id does nothing", "flowchart TD\nA e1@--> B\nstyle e1 fill:#f00", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"edge e1 A B normal none arrow_point \"\"/text",
	}},
	{"classes on edges", "flowchart TD\nA --> B\nA --> B\nA --> B\nclass L_A_B_0,L_A_B_2 c\nclassDef c color:#f00,fill:#0f0", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"edge L_A_B_0 A B normal none arrow_point \"\"/text compiled=color:#f00|fill:#0f0|color:#f00",
		"edge L_A_B_2 A B normal none arrow_point \"\"/text compiled=color:#f00|fill:#0f0|color:#f00",
		"edge L_A_B_3 A B normal none arrow_point \"\"/text",
	}},
	{"linkStyle adds fill:none", "flowchart TD\nA-->B\nA-->C\nlinkStyle 0,1 stroke:#f00", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"node C rect \"C\"/text",
		"edge L_A_B_0 A B normal none arrow_point \"\"/text styles=stroke:#f00|fill:none",
		"edge L_A_C_0 A C normal none arrow_point \"\"/text styles=stroke:#f00|fill:none",
	}},
	{"linkStyle keeps a fill", "flowchart TD\nA-->B\nA-->C\nlinkStyle 0,1 fill:#f00", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"node C rect \"C\"/text",
		"edge L_A_B_0 A B normal none arrow_point \"\"/text styles=fill:#f00",
		"edge L_A_C_0 A C normal none arrow_point \"\"/text styles=fill:#f00",
	}},
	{"linkStyle replaces, default goes first", "flowchart TD\nA-->B\nlinkStyle default stroke:#00f\nlinkStyle 0 stroke:#f00\nlinkStyle 0 stroke:#0f0", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"edge L_A_B_0 A B normal none arrow_point \"\"/text styles=stroke:#00f|stroke:#0f0|fill:none",
	}},
	{"default and node classes", "flowchart TD\nclassDef default fill:#f00\nclassDef node stroke:#00f\nA", []string{
		"TB",
		"node A rect \"A\"/text compiled=fill:#f00|stroke:#00f",
	}},
	{"classes with color, and an escaped comma", "flowchart TD\nclassDef c stroke-dasharray:5\\,5,color:red\nA:::c", []string{
		"TB",
		"node A rect \"A\"/text compiled=stroke-dasharray:5,5|color:red|color:red",
	}},
	{"a class on two ids", "flowchart TD\nA\nB\nclass A,B c\nclassDef c fill:#f00", []string{
		"TB",
		"node A rect \"A\"/text compiled=fill:#f00",
		"node B rect \"B\"/text compiled=fill:#f00",
	}},
	{"a subgraph does not take default", "flowchart TD\nclassDef default fill:#f00\nsubgraph s\nA\nend", []string{
		"TB",
		"group s \"s\"/text",
		"node A rect \"A\"/text in=s compiled=fill:#f00",
	}},
	{"a vertex that is a subgraph lends it its styles", "flowchart TD\ns\nstyle s fill:#f00\nsubgraph s\nA\nend", []string{
		"TB",
		"group s \"s\"/text styles=fill:#f00",
		"node A rect \"A\"/text in=s",
	}},
	{"click adds a class", "flowchart TD\nA\nclick A,B \"u\"\nB\nclassDef clickable fill:#f00", []string{
		"TB",
		"node A rect \"A\"/text compiled=fill:#f00",
		"node B rect \"B\"/text",
	}},
	{"the first subgraph keeps a member", "flowchart TD\nsubgraph a\nA\nend\nsubgraph b\nA\nB\nend", []string{
		"TB",
		"group b \"b\"/text",
		"group a \"a\"/text",
		"node A rect \"A\"/text in=a",
		"node B rect \"B\"/text in=b",
	}},
	{"a repeated subgraph merges into the first", "flowchart TD\nsubgraph s[a]\nA\nend\nsubgraph s[b]\nB\nend", []string{
		"TB",
		"group s \"a\"/text",
		"node A rect \"A\"/text in=s",
		"node B rect \"B\"/text in=s",
	}},
	{"a title with a space has no id", "flowchart TD\nsubgraph a b\nend\nsubgraph a b\nend", []string{
		"TB",
		"group subGraph1 \"a b\"/text",
		"group subGraph0 \"a b\"/text",
	}},
	{"an id with a title", "flowchart TD\nsubgraph \"x y\" [t]\nend", []string{
		"TB",
		"group x y \"t\"/text",
	}},
	{"nested ones come after what holds them", "flowchart TD\nsubgraph a\n subgraph b\n  A\n end\n subgraph c\n  B\n end\nend", []string{
		"TB",
		"group a \"a\"/text",
		"group c \"c\"/text in=a",
		"group b \"b\"/text in=a",
		"node A rect \"A\"/text in=b",
		"node B rect \"B\"/text in=c",
	}},
	{"accTitle is a member", "flowchart TD\nsubgraph a\n A\n accTitle: B\nend\nB", []string{
		"TB",
		"group a \"a\"/text",
		"node A rect \"A\"/text in=a",
		"node B rect \"B\"/text in=a",
	}},
	{"the last direction wins", "flowchart TD\nsubgraph s\n direction LR\n direction BT\n A\nend", []string{
		"TB",
		"group s \"s\"/text dir=BT",
		"node A rect \"A\"/text in=s",
	}},
	{"a direction stays as written", "flowchart TD\nsubgraph s\n direction TD\nend", []string{
		"TB",
		"group s \"s\"/text dir=TD",
	}},
	{"direction at the top does nothing", "flowchart TD\ndirection LR", []string{
		"TB",
	}},
	{"end needs no separator", "flowchart TD\nsubgraph a;A;end", []string{
		"TB",
		"group a \"a\"/text",
		"node A rect \"A\"/text in=a",
	}},
	{"metadata after &", "flowchart TD\nA@{ label: \"x\" } & B@{ shape: cyl }", []string{
		"TB",
		"node A rect \"x\"/markdown",
		"node B cylinder \"B\"/text",
	}},
	{"metadata after a bracket", "flowchart TD\nA[x]@{ label: y }", []string{
		"TB",
		"node A rect \"y\"/markdown",
	}},
	{"a shape by another name", "flowchart TD\nA@{ shape: lean-r, label: \"a b\" }", []string{
		"TB",
		"node A leanright \"a b\"/markdown",
	}},
	{"metadata in a block", "flowchart TD\nA@{\n  label: x\n  shape: cyl\n}", []string{
		"TB",
		"node A cylinder \"x\"/markdown",
	}},
	{"labelType", "flowchart TD\nA@{ label: x, labelType: string }\nB@{ label: x, labelType: foo }", []string{
		"TB",
		"node A rect \"x\"/string",
		"node B rect \"x\"/markdown",
	}},
	{"a word with a colon is a key", "flowchart TD\nA@{ shape:rect, label: x }", []string{
		"TB",
		"node A rect \"x\"/markdown",
	}},
	{"metadata on a subgraph", "flowchart TD\nsubgraph s\nend\ns@{ shape: nope }", []string{
		"TB",
		"group s \"s\"/text",
	}},
	{"curve on an edge", "flowchart TD\nA e1@--> B\ne1@{ curve: basis }", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"edge e1 A B normal none arrow_point \"\"/text",
	}},
	{"a } takes the blanks after it", "flowchart TD\nA[\"a}   \n  b\"]", []string{
		"TB",
		"node A rect \"a}\\n  b\"/string",
	}},
	{"accTitle with nothing after it", "flowchart TD\naccTitle:", []string{
		"TB",
	}},
	{"a dash then > ends an id", "flowchart TD\nA->B]", []string{
		"TB",
		"node A- odd \"B\"/text",
	}},
	{"metadata before & comes after the next vertex", "flowchart TD\nA@{ label: x } & A[y]", []string{
		"TB",
		"node A rect \"x\"/markdown",
	}},
	{"metadata comes before the link", "flowchart TD\nA e1@--> e1@{ label: x }", []string{
		"TB",
		"node A rect \"A\"/text",
		"node e1 rect \"x\"/markdown",
		"edge e1 A e1 normal none arrow_point \"\"/text",
	}},
	{"a line break in metadata", "flowchart TD\nA@{ label: \"a\n  b\" }", []string{
		"TB",
		"node A rect \"a<br/>b\"/markdown",
	}},
	{"only the first @ leaves an id", "flowchart TD\nA a@b@--> B", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"edge ab@ A B normal none arrow_point \"\"/text",
	}},
	{"text is trimmed", "flowchart TD\nA[ x ]\nB-->| t |C", []string{
		"TB",
		"node A rect \"x\"/text",
		"node B rect \"B\"/text",
		"node C rect \"C\"/text",
		"edge L_B_C_0 B C normal none arrow_point \"t\"/text",
	}},
	{"thick at both ends", "flowchart TD\nA<==>B", []string{
		"TB",
		"node A rect \"A\"/text",
		"node B rect \"B\"/text",
		"edge L_A_B_0 A B thick arrow_point arrow_point \"\"/text",
	}},
	{"words that start like keywords", "flowchart TD\nstyleX --> vx", []string{
		"TB",
		"node styleX rect \"styleX\"/text",
		"node vx rect \"vx\"/text",
		"edge L_styleX_vx_0 styleX vx normal none arrow_point \"\"/text",
	}},
	{"end takes the line break after it", "flowchart TD\nsubgraph a end\nx\nend", []string{
		"TB",
		"group subGraph0 \"a end\\nx\"/text",
	}},
	{"a string in a title is text", "flowchart TD\nsubgraph \"q\"\nend", []string{
		"TB",
		"group q \"q\"/text",
	}},
	{"every subgraph counts", "flowchart TD\nsubgraph s\nend\nsubgraph s\nend\nsubgraph a b\nend", []string{
		"TB",
		"group subGraph2 \"a b\"/text",
		"group s \"s\"/text",
	}},
	{"a space in the title of an id", "flowchart TD\nsubgraph a [b c]\nend", []string{
		"TB",
		"group a \"b c\"/text",
	}},
	{"a title is trimmed", "flowchart TD\nsubgraph a[ b ]\nend", []string{
		"TB",
		"group a \"b\"/text",
	}},
	{"a keyword in a title", "flowchart TD\nsubgraph a graph b\nend", []string{
		"TB",
		"group subGraph0 \"a graph b\"/text",
	}},
	{"fill in a text style", "flowchart TD\nclassDef c fill-color:red\nA:::c", []string{
		"TB",
		"node A rect \"A\"/text compiled=fill-color:red|bgFill-color:red",
	}},
	{"compiled styles are trimmed", "flowchart TD\nclassDef c fill:#f00, stroke:#0f0\nA:::c", []string{
		"TB",
		"node A rect \"A\"/text compiled=fill:#f00|stroke:#0f0",
	}},
	{"a class of two names", "flowchart TD\nclassDef a,b fill:#f00\nA:::a\nB:::b", []string{
		"TB",
		"node A rect \"A\"/text compiled=fill:#f00",
		"node B rect \"B\"/text compiled=fill:#f00",
	}},
	{"a blank before the colon of a key", "flowchart TD\nA@{label : x}", []string{
		"TB",
		"node A rect \"x\"/markdown",
	}},
	{"a subgraph is no member of itself", "flowchart TD\nsubgraph s\ns --> B\nend", []string{
		"TB",
		"group s \"s\"/text",
		"node B rect \"B\"/text in=s",
		"edge L_s_B_0 s B normal none arrow_point \"\"/text",
	}},
	{"a group has no ellipse", "flowchart TD\ns(-x-)\nsubgraph s\nend", []string{
		"TB",
		"group s \"s\"/text",
	}},
	{"an empty shape changes nothing", "flowchart TD\nA@{ shape: \"\" }", []string{
		"TB",
		"node A rect \"A\"/text",
	}},
	{"quotes in a trapezoid", "flowchart TD\nA[/ \"x\"/]\nB[\\ \"y\" \\]\nC[/ \"/]\nD[/ \"a\" b/]", []string{
		"TB",
		"node A leanright \"x\"/text",
		"node B leanleft \"y\"/text",
		"node C leanright \"\\\"\"/text",
		"node D leanright \"\\\"a\\\" b\"/text",
	}},
	{"a tab after a colon", "flowchart TD\nA@{ shape:\tcircle }", []string{
		"TB",
		"node A circle \"A\"/text",
	}},
	{"a left arrow", "flowchart >\nA", []string{
		"LR",
		"node A rect \"A\"/text",
	}},
	{"a caret", "graph ^\nA", []string{
		"BT",
		"node A rect \"A\"/text",
	}},
	{"a v", "graph v\nA", []string{
		"TB",
		"node A rect \"A\"/text",
	}},
	{"BR", "flowchart BR\nA", []string{
		"BR",
		"node A rect \"A\"/text",
	}},
	{"no direction", "graph\nA", []string{
		"TB",
		"node A rect \"A\"/text",
	}},
	{"elk", "flowchart-elk LR\nA", []string{
		"LR",
		"node A rect \"A\"/text",
	}},
}

func TestRead(t *testing.T) {
	for _, c := range readCases {
		t.Run(c.name, func(t *testing.T) {
			body, _ := syntax.Prepare(c.src)
			d, err := Read(body)
			if err != nil {
				t.Fatal(err)
			}
			if got := dump(d); !slices.Equal(got, c.want) {
				t.Errorf("got\n  %s\nwant\n  %s", strings.Join(got, "\n  "), strings.Join(c.want, "\n  "))
			}
		})
	}
}

var shapeNames = []string{
	"rect", "round", "stadium", "subroutine", "cylinder", "circle", "doublecircle", "diamond",
	"hexagon", "odd", "trapezoid", "invtrapezoid", "leanright", "leanleft",
}

// dump writes d a line for each thing in it, with the fields that are set.
func dump(d *Diagram) []string {
	out := []string{d.Direction}
	for _, n := range d.Nodes {
		s := "node " + n.ID + " " + shapeNames[n.Shape]
		if n.Group {
			s = "group " + n.ID
		}
		s += fmt.Sprintf(" %q/%s", n.Label.Text, n.Label.Kind)
		if n.Parent != "" {
			s += " in=" + n.Parent
		}
		if n.Dir != "" {
			s += " dir=" + n.Dir
		}
		out = append(out, s+dumpStyles(n.Styles, n.Compiled))
	}
	for _, e := range d.Edges {
		s := fmt.Sprintf("edge %s %s %s %s %s %s %q/%s", e.ID, e.Start, e.End, e.Stroke, e.StartMarker, e.EndMarker, e.Label.Text, e.Label.Kind)
		out = append(out, s+dumpStyles(e.Styles, e.Compiled))
	}
	return out
}

func dumpStyles(styles, compiled []string) string {
	s := ""
	if len(styles) > 0 {
		s += " styles=" + strings.Join(styles, "|")
	}
	if len(compiled) > 0 {
		s += " compiled=" + strings.Join(compiled, "|")
	}
	return s
}

func links(n int) string {
	var b strings.Builder
	b.WriteString("flowchart TD\n")
	for i := range n {
		fmt.Fprintf(&b, " n%d --> n%d\n", i, i+1)
	}
	return b.String()
}

func TestRefused(t *testing.T) {
	for _, c := range []struct {
		name, src   string
		line, col   int
		unsupported bool
	}{
		{"no = in a thick link's text", "flowchart TD\nA == x=y ==> B", 2, 7, false},
		{"no . in a dotted link's text", "flowchart TD\nA -. x.y .-> B", 2, 7, false},
		{"no = in an id", "flowchart TD\nA=B", 2, 2, false},
		{"end is a keyword", "flowchart TD\nA --> end", 2, 7, false},
		{"a link needs its end", "flowchart TD\nA --B", 3, 1, false},
		{"an open bracket", "flowchart TD\nA[x", 3, 1, false},
		{"a second header", "flowchart TD\nA --> graph", 2, 7, false},
		{"a subgraph not closed", "flowchart TD\nsubgraph a\nA", 4, 1, false},
		{"lines after a } count", "flowchart TD\nA{x}\n  \n  \n  §", 5, 3, false},
		{"a subgraph with no title", "flowchart TD\nsubgraph\nend", 2, 1, false},
		{"subgraphs inside each other", "flowchart TD\nsubgraph a\n b\nend\nsubgraph b\n a\nend", 5, 1, false},
		{"no such link", "flowchart TD\nA-->B\nlinkStyle 0,1 stroke:#f00", 3, 13, false},
		{"a link number with a 0 before it", "flowchart TD\nA-->B\nlinkStyle 00 stroke:#f00", 3, 11, false},
		{"no such link for a curve", "flowchart TD\nA-->B\nlinkStyle 1 interpolate basis", 3, 11, false},
		{"an ellipse has no shape to draw", "flowchart TD\nA(-x-)", 2, 1, false},
		{"too many links", links(maxEdges + 1), maxEdges + 2, 6, false},
		{"-- ends a link's text", "flowchart TD\nA -- a--b --> C", 2, 7, false},
		{"~~~ does not end a link's text", "flowchart TD\nA -- t ~~~ B", 3, 1, false},
		{"a letter newer than Mermaid's table", "flowchart TD\n鿍 --> A", 2, 1, false},
		{"text then a string", "flowchart TD\nA[a\"b\"]", 2, 5, false},
		{"& needs a space before", "flowchart TD\nA[x]& B", 2, 5, false},
		{"& needs a space after", "flowchart TD\nA &B", 2, 4, false},
		{"a space then ; after the direction", "flowchart TD ;A", 1, 14, false},
		{"a space and no style", "flowchart TD\nA-->B\nlinkStyle 0 interpolate basis ", 3, 31, false},
		{"a } before blank lines", "flowchart TD\nA}  \n\n\nB", 2, 2, false},
		{"a key given twice", "flowchart TD\nA@{ shape: rect, shape: cyl }", 2, 2, false},
		{"a shape to come", "flowchart TD\nA@{ shape: card }", 2, 2, true},
		{"a shape Mermaid has not got", "flowchart TD\nA@{ shape: nope }", 2, 2, true},
		{"an icon", "flowchart TD\nA@{ icon: x }", 2, 2, true},
		{"YAML past keys and scalars", "flowchart TD\nA@{ label: [a] }", 2, 2, true},
		{"a control character in metadata", "flowchart TD\nA@{ label: a\x01b }", 2, 2, false},
		{"a comment after a tab", "flowchart TD\nA@{ label: x\t# c }", 2, 2, true},
		{"a bracket in a plain value", "flowchart TD\nA@{ label: a[b] }", 2, 2, true},
		{"a backslash in a string", "flowchart TD\nA@{ label: \"x\\ny\" }", 2, 2, true},
		{"a block with two indents", "flowchart TD\nA@{\n  label: x\n    shape: cyl\n}", 2, 2, true},
		{"a quoted key with no blank after it in a block", "flowchart TD\nA@{\n  \"label\":x\n}", 2, 2, true},
		{"nothing in a block", "flowchart TD\nA@{\n}", 2, 2, true},
		{"a number for a label", "flowchart TD\nA@{ label: 5 }", 2, 2, true},
		{"animation", "flowchart TD\nA e1@--> B\ne1@{ animate: true }", 3, 3, true},
		{"a collapsed subgraph", "flowchart TD\nsubgraph s\nend\ns@{ view: collapsed }", 4, 2, true},
		{"HTML in a label", "flowchart TD\nA[\"<b>x</b>\"]", 2, 1, true},
		{"a label given after its node", "flowchart TD\nA\nB --> A[\"fa:fa-car\"]", 3, 7, true},
		{"HTML on an edge", "flowchart TD\nA -->|\"<b>x</b>\"| B", 2, 2, true},
		{"HTML in a title", "flowchart TD\n\nsubgraph s[\"<i>x</i>\"]\nend", 3, 1, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			body, _ := syntax.Prepare(c.src)
			_, err := Read(body)
			var e *syntax.Error
			if !errors.As(err, &e) {
				t.Fatalf("got %v", err)
			}
			if e.Line != c.line || e.Col != c.col || e.IsUnsupported() != c.unsupported {
				t.Errorf("got %v", err)
			}
		})
	}
}

// A later bracket takes an ellipse's shape away, and Mermaid draws what is left.
func TestEllipseReplaced(t *testing.T) {
	body, _ := syntax.Prepare("flowchart TD\nA(-x-)\nA[y]")
	if _, err := Read(body); err != nil {
		t.Fatal(err)
	}
}
