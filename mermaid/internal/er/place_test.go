package er

import (
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

func placed(t *testing.T, src string) (Placement, error) {
	t.Helper()
	body, _ := syntax.Prepare("erDiagram\n" + src)
	d, err := Read(body)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return Place(d, scene.Arial)
}

// The places are what Chrome measured of Mermaid 12.3.0's ELK layout.
func TestPlaceAsChromeMeasuredIt(t *testing.T) {
	for _, c := range []struct {
		name, src string
		centers   map[string][2]float64
		line      [][2]float64
		text      [2]float64
	}{
		{
			"two entities and a line", "A ||--o{ B : x",
			map[string][2]float64{"entity-A-0": {62, 52.5}, "entity-B-1": {62, 234.5}},
			[][2]float64{{62, 93}, {62, 194}}, [2]float64{62, 143.5},
		},
		{
			"a line from a subgraph", "subgraph S\n A\nend\nS ||--o{ B : x",
			map[string][2]float64{"S": {86, 88.5}, "entity-A-0": {86, 100.5}, "entity-B-2": {86, 306.5}},
			[][2]float64{{86, 165}, {86, 266}}, [2]float64{86, 215.5},
		},
	} {
		p, err := placed(t, c.src)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		for _, n := range p.R.Nodes {
			if want := c.centers[n.ID]; !near(n.X, want[0]) || !near(n.Y, want[1]) {
				t.Errorf("%s: %s at %v %v, want %v", c.name, n.ID, n.X, n.Y, want)
			}
		}
		e := p.R.Edges[0]
		if len(e.Points) != len(c.line) {
			t.Fatalf("%s: line %v, want %v", c.name, e.Points, c.line)
		}
		for i, q := range e.Points {
			if !near(q.X, c.line[i][0]) || !near(q.Y, c.line[i][1]) {
				t.Errorf("%s: line %v, want %v", c.name, e.Points, c.line)
			}
		}
		if !near(e.Label.X, c.text[0]) || !near(e.Label.Y, c.text[1]) {
			t.Errorf("%s: text at %v, want %v", c.name, *e.Label, c.text)
		}
	}
}

func TestPlaceErrors(t *testing.T) {
	for _, c := range []struct {
		name, src   string
		unsupported bool
	}{
		{"two subgraphs with one name", "subgraph S\n A\nend\nsubgraph S\n B\nend", true},
		{"a line to an entity a subgraph replaced", "S ||--o{ B : x\nsubgraph S\n A\nend", false},
		{"a line ELK refuses", "subgraph S\n direction TB\n subgraph T\n  A\n end\nend\nA ||--o{ T : x", false},
	} {
		_, err := placed(t, c.src)
		e, ok := err.(*syntax.Error)
		if !ok || e.IsUnsupported() != c.unsupported {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}
