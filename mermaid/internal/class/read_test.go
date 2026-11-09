package class

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

var kindNames = map[Kind]string{Class: "class", Namespace: "namespace", Note: "note", Interface: "interface"}

func dumpMembers(what string, ms []Member) []string {
	var out []string
	for _, m := range ms {
		out = append(out, fmt.Sprintf("%s %q %q %q %q %q %q", what, m.Visibility, m.ID, m.Parameters, m.ReturnType, m.Classifier, m.Text))
	}
	return out
}

func dump(d *Diagram) []string {
	out := []string{d.Direction}
	for _, n := range d.Nodes {
		s := fmt.Sprintf("node %s %s %q", n.ID, kindNames[n.Kind], n.Label)
		if n.Parent != "" {
			s += " in " + n.Parent
		}
		if len(n.Annotations) > 0 {
			s += " <<" + strings.Join(n.Annotations, ",") + ">>"
		}
		if n.Kind == Class && n.Classes != "default" {
			s += " css " + n.Classes
		}
		if len(n.Styles) > 0 {
			s += " style " + strings.Join(n.Styles, ",")
		}
		out = append(append(append(out, s), dumpMembers("attribute", n.Members)...), dumpMembers("method", n.Methods)...)
	}
	for _, e := range d.Edges {
		out = append(out, fmt.Sprintf("edge %s %s %s %s %s %s %q %q %q", e.ID, e.Start, e.End, e.StartMarker, e.EndMarker, e.Pattern, e.Label, e.StartLabel, e.EndLabel))
	}
	return out
}

// readCases read as they do in Mermaid v12.3.0: each want is what its getData gave, and a
// nil want is an error.
var readCases = []struct {
	name, src string
	want      []string
}{
	{"the numbers at a line's ends", "classDiagram\nA \"1\" --> \"2\" B", []string{
		"TB",
		`node A class "A"`,
		`node B class "B"`,
		`edge id_A_B_1 A B none dependency solid "" "1" "2"`,
	}},
	{"a dotted line, an end each side", "classDiagram\nA <..> B", []string{
		"TB",
		`node A class "A"`,
		`node B class "B"`,
		`edge id_A_B_1 A B dependency dependency dashed "" "" ""`,
	}},
	{"a lollipop with nothing facing it makes an interface", "classDiagram\na ()..b", []string{
		"TB",
		`node b class "b"`,
		`node interface0 interface "a"`,
		`edge id_interface0_b_1 interface0 b lollipop none dashed "" "" ""`,
	}},
	{"a lollipop with an end facing it stays between classes", "classDiagram\na <|--() b", []string{
		"TB",
		`node a class "a"`,
		`node b class "b"`,
		`edge id_a_b_1 a b extension lollipop solid "" "" ""`,
	}},
	{"a method and its generics", "classDiagram\nA : a(b~T~, c~U~) d~V~", []string{
		"TB",
		`node A class "A"`,
		`method "" "a" "b~T~, c~U~" "d~V~" "" "a(b&lt;T&gt;, c&lt;U&gt;) : d&lt;V&gt;"`,
	}},
	{"an attribute's visibility and classifier", "classDiagram\nA : -x$", []string{
		"TB",
		`node A class "A"`,
		`attribute "-" "x" "" "" "$" "\\-x"`,
	}},
	{"a generic class", "classDiagram\nclass A~T,U~", []string{
		"TB",
		`node A class "A&lt;T,U&gt;"`,
	}},
	{"names written apart are one", "classDiagram\nclass Foo Bar", []string{
		"TB",
		`node FooBar class "FooBar"`,
	}},
	{"a dotted namespace makes its ancestors", "classDiagram\nnamespace A.B {\n class X\n}", []string{
		"TB",
		`node A namespace "A"`,
		`node A.B namespace "B" in A`,
		`node X class "X" in A.B`,
	}},
	{"a nested namespace takes the outer name", "classDiagram\nnamespace A {\n namespace B {\n class X\n }\n}", []string{
		"TB",
		`node A namespace "A"`,
		`node A.B namespace "B" in A`,
		`node X class "X" in A.B`,
	}},
	{"a note's line", "classDiagram\nclass A\nnote for A \"t\"", []string{
		"TB",
		`node A class "A"`,
		`node note0 note "t"`,
		`edge edgeNote0 note0 A none none dotted "" "" ""`,
	}},
	{"classDef reaches the classes there already", "classDiagram\nclass A:::hot\nclassDef hot fill:#f00", []string{
		"TB",
		`node A class "A" css default hot style fill:#f00`,
	}},
	{"and only those", "classDiagram\nclassDef hot fill:#f00\nclass A:::hot", []string{
		"TB",
		`node A class "A" css default hot`,
	}},
	{"an annotation needs its class", "classDiagram\n<<i>> Foo", nil},
	{"a method needs a name", "classDiagram\nA : ()", nil},
	{"only one number before a line", "classDiagram\nA \"1\" \"2\" --> B", nil},
}

func TestRead(t *testing.T) {
	for _, c := range readCases {
		t.Run(c.name, func(t *testing.T) {
			body, _ := syntax.Prepare(c.src)
			d, err := Read(body)
			if c.want == nil {
				if err == nil {
					t.Fatalf("read as %q, want an error", dump(d))
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := dump(d); !slices.Equal(got, c.want) {
				t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(c.want, "\n"))
			}
		})
	}
}

// A < in what is parsed goes through DOMPurify, which sumi does not, so it is refused.
func TestReadRefusesAngle(t *testing.T) {
	for _, src := range []string{"classDiagram\nA : List<int> x", "classDiagram\nclass A[\"a<b\"]", "classDiagram\nclass `a<b`"} {
		_, err := Read(src)
		var e *syntax.Error
		if !errors.As(err, &e) || !e.IsUnsupported() {
			t.Errorf("%q: %v, want unsupported", src, err)
		}
	}
}
