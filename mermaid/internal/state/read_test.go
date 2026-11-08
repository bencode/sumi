package state

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// readCases read as they do in Mermaid v12.3.0: each want is what its getData gave. A nil
// want is an error, as Mermaid's, or for a state inside itself sumi's own.
var readCases = []struct {
	name, src string
	want      []string
}{
	{"a machine", "stateDiagram-v2\n[*] --> A\nA --> B : go\nB --> [*]", []string{
		"TB",
		"node root_start start \"root_start\"",
		"node A rect \"A\"",
		"node B rect \"B\"",
		"node root_end end \"root_end\"",
		"edge edge0 root_start A \"\"",
		"edge edge1 A B \"go\"",
		"edge edge2 B root_end \"\"",
	}},
	{"a start and an end for each body", "stateDiagram-v2\nstate X {\n [*] --> A\n A --> [*]\n}\n[*] --> X", []string{
		"TB",
		"node X composite \"X\" group TB",
		"node X_start start \"X_start\" in X",
		"node A rect \"A\" in X",
		"node X_end end \"X_end\" in X",
		"node root_start start \"root_start\"",
		"edge edge0 X_start A \"\"",
		"edge edge1 A X_end \"\"",
		"edge edge2 root_start X \"\"",
	}},
	{"choice, fork, join and a note", "stateDiagram-v2\nstate C <<choice>>\nstate F <<fork>>\nstate J <<join>>\nA --> C\nC --> F\nF --> J\nA\nnote right of A : t", []string{
		"TB",
		"node C choice \"C\"",
		"node F fork \"F\"",
		"node J join \"J\"",
		"node A rect \"A\"",
		"node A----parent noteGroup \"t\" group \"right of\"",
		"node A----note-3 note \"t\" in A----parent \"right of\"",
		"edge edge0 A C \"\"",
		"edge edge1 C F \"\"",
		"edge edge2 F J \"\"",
		"edge A-A----note-3 A A----note-3 \"\" note",
	}},
	{"regions", "stateDiagram-v2\nstate K {\n [*] --> A\n --\n [*] --> B\n}", []string{
		"TB",
		"node K composite \"K\" group TB",
		"node divider-id-1 divider \"\" in K group TB alt",
		"node divider-id-1_start start \"divider-id-1_start\" in divider-id-1",
		"node A rect \"A\" in divider-id-1",
		"node region1 divider \"\" in K group TB alt",
		"node region1_start start \"region1_start\" in region1",
		"node B rect \"B\" in region1",
		"edge edge0 divider-id-1_start A \"\"",
		"edge edge1 region1_start B \"\"",
	}},
	{"nothing after the last -- leaves the body whole", "stateDiagram-v2\nstate X {\n A\n --\n}", []string{
		"TB",
		"node X composite \"X\" group TB",
		"node A rect \"A\" in X",
		"node divider-id-1 divider \"\" in X",
	}},
	{"every other level of groups is alt", "stateDiagram-v2\nstate X {\n state Y {\n  state Z {\n   A\n  }\n }\n}", []string{
		"TB",
		"node X composite \"X\" group TB",
		"node Y composite \"Y\" in X group TB alt",
		"node Z composite \"Z\" in Y group TB",
		"node A rect \"A\" in Z",
	}},
	{"a body keeps its direction", "stateDiagram-v2\nstate X {\n direction LR\n A\n}", []string{
		"TB",
		"node X composite \"X\" group LR",
		"node A rect \"A\" in X",
	}},
	{"a region takes the direction written in it", "stateDiagram-v2\nstate X {\n A\n --\n direction LR\n B\n}", []string{
		"TB",
		"node X composite \"X\" group TB",
		"node divider-id-1 divider \"\" in X group TB alt",
		"node A rect \"A\" in divider-id-1",
		"node region1 divider \"\" in X group LR alt",
		"node B rect \"B\" in region1",
	}},
	{"a line that mentions a direction is one", "stateDiagram-v2\nA --> B direction LR", []string{
		"LR",
	}},
	{"state without a body sets nothing", "stateDiagram-v2\nstate Foo", []string{
		"TB",
	}},
	{"a floating note sets nothing", "stateDiagram-v2\nnote \"x\" as n", []string{
		"TB",
	}},
	{"a note first names a state with no shape", "stateDiagram-v2\nnote left of A : t", []string{
		"TB",
		"node A----parent noteGroup \"t\" group \"left of\"",
		"node A----note-0 note \"t\" in A----parent \"left of\"",
		"node A none \"A\"",
		"edge A----note-0-A A----note-0 A \"\" note",
	}},
	{"a left note leads to its state", "stateDiagram-v2\nA\nnote left of A : t", []string{
		"TB",
		"node A rect \"A\"",
		"node A----parent noteGroup \"t\" group \"left of\"",
		"node A----note-0 note \"t\" in A----parent \"left of\"",
		"edge A----note-0-A A----note-0 A \"\" note",
	}},
	{"a note in a body stays out of it", "stateDiagram-v2\nstate X {\n A\n note right of A : t\n}", []string{
		"TB",
		"node X composite \"X\" group TB",
		"node A rect \"A\" in X",
		"node A----parent noteGroup \"t\" group \"right of\"",
		"node A----note-0 note \"t\" in A----parent \"right of\"",
		"edge A-A----note-0 A A----note-0 \"\" note",
	}},
	{"more notes on a state", "stateDiagram-v2\nA\nnote right of A : t\nnote left of A : u", []string{
		"TB",
		"node A rect \"A\"",
		"node A----parent noteGroup \"u\" group \"left of\"",
		"node A----note-0 note \"t\" in A----parent \"right of\"",
		"node A----note-1 note \"u\" in A----parent \"left of\"",
		"edge A-A----note-0 A A----note-0 \"\" note",
		"edge A----note-1-A A----note-1 A \"\" note",
	}},
	{"LEFT OF is not left of", "stateDiagram-v2\nA\nnote LEFT OF A : t", []string{
		"TB",
		"node A rect \"A\"",
		"node A----parent noteGroup \"t\" group \"LEFT OF\"",
		"node A----note-0 note \"t\" in A----parent \"LEFT OF\"",
		"edge A-A----note-0 A A----note-0 \"\" note",
	}},
	{"a note of several lines", "stateDiagram-v2\nA\nnote left of A\n t\n u\nend note", []string{
		"TB",
		"node A rect \"A\"",
		"node A----parent noteGroup \"t\\n u\" group \"left of\"",
		"node A----note-0 note \"t\\n u\" in A----parent \"left of\"",
		"edge A----note-0-A A----note-0 A \"\" note",
	}},
	{"the first two characters of a note go", "stateDiagram-v2\nA\nnote left of A :t", []string{
		"TB",
		"node A rect \"A\"",
		"node A----parent noteGroup \"t\" group \"left of\"",
		"node A----note-0 note \"t\" in A----parent \"left of\"",
		"edge A----note-0-A A----note-0 A \"\" note",
	}},
	{"a second description makes a title", "stateDiagram-v2\nA: one\nA: two\nA: three", []string{
		"TB",
		"node A titled \"one\" \"two\" \"three\"",
	}},
	{"one description is a label", "stateDiagram-v2\nA: one", []string{
		"TB",
		"node A rect \"one\"",
	}},
	{"state as with a colon", "stateDiagram-v2\nstate \"d\" as A:e", []string{
		"TB",
		"node A titled \"d\" \"e\"",
	}},
	{"a group takes one description", "stateDiagram-v2\nstate \"d\" as A {\n B\n}\nA: more", nil},
	{"a state in two bodies stays where it first was, in the last", "stateDiagram-v2\nstate X {\n A\n}\nstate Y {\n A\n}", []string{
		"TB",
		"node X composite \"X\" group TB",
		"node A rect \"A\" in Y",
		"node Y composite \"Y\" group TB",
	}},
	{"the first mention sets the shape", "stateDiagram-v2\nA --> B\nstate A <<choice>>", []string{
		"TB",
		"node A rect \"A\"",
		"node B rect \"B\"",
		"edge edge0 A B \"\"",
	}},
	{"a state of the id root is the root", "stateDiagram-v2\nstate root {\n A\n}", []string{
		"TB",
		"node A rect \"A\"",
	}},
	{"a [*] named by state as", "stateDiagram-v2\nstate \"x\" as [*] {\n [*] --> A\n}", []string{
		"TB",
		"node root_start composite \"x\" group TB",
		"node [*]_start start \"[*]_start\" in root_start",
		"node A rect \"A\" in root_start",
		"edge edge0 [*]_start A \"\"",
	}},
	{"A:::c alone does nothing", "stateDiagram-v2\nclassDef c fill:#f00\nA:::c", []string{
		"TB",
		"node A rect \"A\"",
	}},
	{"A:::c on a transition", "stateDiagram-v2\nclassDef c fill:#f00\nA:::c --> B", []string{
		"TB",
		"node A rect \"A\" compiled \"fill:#f00\"",
		"node B rect \"B\"",
		"edge edge0 A B \"\"",
	}},
	{"what a body says of classes does nothing", "stateDiagram-v2\nclassDef c fill:#f00\nstate X {\n A:::c --> B\n classDef d fill:#0f0\n class A c\n style A fill:#00f\n}", []string{
		"TB",
		"node X composite \"X\" group TB",
		"node A rect \"A\" in X",
		"node B rect \"B\" in X",
		"edge edge0 A B \"\"",
	}},
	{"a style replaces the last", "stateDiagram-v2\nstyle A fill:#f00\nstyle A stroke:#0f0\nA", []string{
		"TB",
		"node A rect \"A\" styles \"stroke:#0f0\"",
	}},
	{"a classDef loses each first ;", "stateDiagram-v2\nclassDef c fill:#f00,color:#fff;\nclass A c\nA", []string{
		"TB",
		"node A rect \"A\" compiled \"fill:#f00,color:#fff\"",
	}},
	{"several classes", "stateDiagram-v2\nclass A c d\nclassDef c fill:#f00\nclassDef d stroke:#0f0\nA", []string{
		"TB",
		"node A rect \"A\" compiled \"fill:#f00,stroke:#0f0\"",
	}},
	{"words that are almost keywords", "stateDiagram-v2\nclicked --> A", []string{
		"TB",
		"node clicked rect \"clicked\"",
		"node A rect \"A\"",
		"edge edge0 clicked A \"\"",
	}},
	{"a colon in a transition", "stateDiagram-v2\nA --> B : x::y", nil},
	{"a comment after an id", "stateDiagram-v2\nA%%c", []string{
		"TB",
		"node A rect \"A\"",
	}},
	{"two statements on a line", "stateDiagram-v2\nA B", []string{
		"TB",
		"node A rect \"A\"",
		"node B rect \"B\"",
	}},
	{"classDef default is refused", "stateDiagram-v2\nclassDef default fill:#f00\nA", nil},
	{"a name of two words is refused", "stateDiagram-v2\nstate A b {\n}", nil},
	{"-- at the root is refused", "stateDiagram-v2\n--", nil},
	{"a state inside itself is refused", "stateDiagram-v2\nstate A {\n state B {\n  A\n }\n}", nil},
}

var shapeWords = map[Shape]string{
	NoShape: "none", Rect: "rect", RectWithTitle: "titled", Start: "start", End: "end", Fork: "fork", Join: "join",
	Choice: "choice", Note: "note", Composite: "composite", Divider: "divider", NoteGroup: "noteGroup",
}

// render writes d as the wants do, the regions Mermaid gives random ids numbered.
func render(d *Diagram) []string {
	regions := map[string]string{}
	name := func(s string) string {
		if head, rest, ok := strings.Cut(s, "\n"); ok && head == "divider" {
			pos, suffix, _ := strings.Cut(rest, "_")
			if suffix != "" {
				suffix = "_" + suffix
			}
			key := head + pos
			if regions[key] == "" {
				regions[key] = "region" + strconv.Itoa(len(regions)+1)
			}
			return regions[key] + suffix
		}
		return s
	}
	out := []string{d.Direction}
	for _, n := range d.Nodes {
		l := "node " + name(n.ID) + " " + shapeWords[n.Shape] + " " + strconv.Quote(name(n.Label))
		for _, s := range n.Descriptions {
			l += " " + strconv.Quote(s)
		}
		if n.Parent != "" {
			l += " in " + name(n.Parent)
		}
		if n.Group {
			l += " group"
		}
		if n.Dir != "" {
			l += " " + n.Dir
		}
		if n.Alt {
			l += " alt"
		}
		if n.Position != "" {
			l += " " + strconv.Quote(n.Position)
		}
		if len(n.Styles) > 0 {
			l += " styles " + strconv.Quote(strings.Join(n.Styles, ","))
		}
		if len(n.Compiled) > 0 {
			l += " compiled " + strconv.Quote(strings.Join(n.Compiled, ","))
		}
		out = append(out, l)
	}
	for _, e := range d.Edges {
		l := "edge " + name(e.ID) + " " + name(e.Start) + " " + name(e.End) + " " + strconv.Quote(e.Label)
		if e.Note {
			l += " note"
		}
		out = append(out, l)
	}
	return out
}

func TestRead(t *testing.T) {
	for _, c := range readCases {
		t.Run(c.name, func(t *testing.T) {
			body, _ := syntax.Prepare(c.src)
			d, err := Read(body)
			if c.want == nil {
				if err == nil {
					t.Fatalf("read as %q, want an error", render(d))
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := render(d); !slices.Equal(got, c.want) {
				t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(c.want, "\n"))
			}
		})
	}
}
