package er

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

func dump(d *Diagram) []string {
	out := []string{d.Direction}
	for _, n := range d.Nodes {
		s := fmt.Sprintf("%s %q in %q css %q compiled %q styles %q", n.ID, n.Label, n.Parent, n.Classes, n.Compiled, n.Styles)
		if n.Group {
			s += " group " + n.Dir
		}
		if n.Alias != "" {
			s += " alias " + n.Alias
		}
		out = append(out, s)
		for _, a := range n.Attributes {
			out = append(out, fmt.Sprintf("attribute %s %s %s %q", a.Type, a.Name, strings.Join(a.Keys, ","), a.Comment))
		}
	}
	for _, e := range d.Edges {
		out = append(out, fmt.Sprintf("edge %s %s %s %s %s %s %q", e.ID, e.Start, e.End, e.StartMarker, e.EndMarker, e.Pattern, e.Label))
	}
	return out
}

func entity(id, label, parent, css, compiled, styles string) string {
	return fmt.Sprintf("%s %q in %q css %q compiled %s styles %s", id, label, parent, css, compiled, styles)
}

// readCases read as they do in Mermaid v12.3.0: each want is what its getData gave.
var readCases = []struct {
	name, src string
	want      []string
}{
	{"a relationship's markers are those beside each end", "A ||--o{ B : x", []string{"TB",
		entity("entity-A-0", "A", "", "default", "[]", "[]"),
		entity("entity-B-1", "B", "", "default", "[]", "[]"),
		`edge id_entity-A-0_entity-B-1_0 entity-A-0 entity-B-1 only_one zero_or_more solid "x"`,
	}},
	{"attributes", "A {\n int x PK \"c\"\n}", []string{"TB",
		entity("entity-A-0", "A", "", "default", "[]", "[]"),
		`attribute int x PK "c"`,
	}},
	{"a direction anywhere on a line takes the line", "A ||--o{ B : direction TB", []string{"TB"}},
	{"the first alias stays", "A[\"x\"]\nA[\"y\"]", []string{"TB",
		entity("entity-A-0", "A", "", "default", "[]", "[]") + " alias x",
	}},
	{"a style reaches only what exists by then", "style A fill:#f00\nA", []string{"TB",
		entity("entity-A-0", "A", "", "default", "[]", "[]"),
	}},
	{"a classDef of default reaches every entity", "classDef default fill:#f00\nA", []string{"TB",
		entity("entity-A-0", "A", "", "default", `["fill:#f00"]`, "[]"),
	}},
	{"a color is kept again for the text", "classDef c fill:#f00,color:#00f\nA:::c", []string{"TB",
		entity("entity-A-0", "A", "", "default c", `["fill:#f00" "color:#00f" "color:#00f"]`, "[]"),
	}},
	{"subgraphs come last to first, before the entities", "subgraph O\n A\n subgraph I\n  B\n end\nend", []string{"TB",
		entity("O", "O", "", "", "[]", "[]") + " group ",
		entity("I", "I", "O", "", "[]", "[]") + " group ",
		entity("entity-A-0", "A", "O", "default", "[]", "[]"),
		entity("entity-B-1", "B", "I", "default", "[]", "[]"),
	}},
	{"a relationship joins a subgraph that has closed", "subgraph S\n A\nend\nS ||--o{ B : x", []string{"TB",
		entity("S", "S", "", "", "[]", "[]") + " group ",
		entity("entity-A-0", "A", "S", "default", "[]", "[]"),
		entity("entity-B-2", "B", "", "default", "[]", "[]"),
		`edge id_S_entity-B-2_0 S entity-B-2 only_one zero_or_more solid "x"`,
	}},
	{"a keyword in a subgraph names what it says", "\"classDef\"\nsubgraph S\n classDef c fill:#f00\nend", []string{"TB",
		entity("S", "S", "", "", "[]", "[]") + " group ",
		entity("entity-classDef-0", "classDef", "S", "default", "[]", "[]"),
	}},
}

func TestRead(t *testing.T) {
	for _, c := range readCases {
		t.Run(c.name, func(t *testing.T) {
			body, _ := syntax.Prepare("erDiagram\n" + c.src)
			d, err := Read(body)
			if err != nil {
				t.Fatal(err)
			}
			if got := dump(d); !slices.Equal(got, c.want) {
				t.Errorf("\n got %q\nwant %q", got, c.want)
			}
		})
	}
}

func TestReadErrors(t *testing.T) {
	cases := []struct {
		name, src   string
		line, col   int
		unsupported bool
	}{
		{"a subgraph inside itself", "erDiagram\nsubgraph A\n A\nend", 2, 1, false},
		{"subgraphs inside each other", "erDiagram\nsubgraph A\n B\nend\nsubgraph B\n A\nend", 5, 1, false},
		{"a subgraph not closed", "erDiagram\nsubgraph A\n B", 4, 1, false},
		{"a relationship without its text", "erDiagram\nA ||--o{ B", 2, 11, false},
		{"markup in a subgraph's title", "erDiagram\nsubgraph \"a<b\"\n A\nend", 2, 1, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body, _ := syntax.Prepare(c.src)
			_, err := Read(body)
			var e *syntax.Error
			if !errors.As(err, &e) {
				t.Fatalf("got %v, want a syntax error", err)
			}
			if e.Line != c.line || e.Col != c.col || e.IsUnsupported() != c.unsupported {
				t.Errorf("got %v at %d:%d unsupported %v, want %d:%d unsupported %v", e, e.Line, e.Col, e.IsUnsupported(), c.line, c.col, c.unsupported)
			}
		})
	}
}
