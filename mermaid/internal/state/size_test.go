package state

import (
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

func sized(t *testing.T, src string) (*Diagram, []Sized, error) {
	t.Helper()
	body, _ := syntax.Prepare(src)
	d, err := Read(body)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	s, err := Size(d, scene.Arial)
	return d, s, err
}

func TestSizeRefusesWhatHasNoShape(t *testing.T) {
	for _, src := range []string{
		"stateDiagram-v2\nnote left of x : only a note names x",
		"stateDiagram-v2\nstate a {\nb\n--\n}",
	} {
		if _, _, err := sized(t, src); err == nil || err.(*syntax.Error).IsUnsupported() {
			t.Errorf("%q: got %v, want an error of the source", src, err)
		}
	}
}

// A bar runs across the flow, and the flow is the nearest group's, which is top to
// bottom when the group says nothing, whatever the diagram's is.
func TestForkRunsAcrossTheNearestFlow(t *testing.T) {
	for _, c := range []struct {
		name, src string
		w, h      float64
	}{
		{"root", "stateDiagram-v2\nstate f <<fork>>", 70, 10},
		{"root LR", "stateDiagram-v2\ndirection LR\nstate f <<fork>>", 10, 70},
		{"plain group in LR", "stateDiagram-v2\ndirection LR\nstate g {\nstate f <<fork>>\n}", 70, 10},
		{"LR group in TB", "stateDiagram-v2\nstate g {\ndirection LR\nstate f <<join>>\n}", 10, 70},
		{"plain group in an LR group", "stateDiagram-v2\nstate g {\ndirection LR\nstate k {\nstate f <<fork>>\n}\n}", 70, 10},
	} {
		d, s, err := sized(t, c.src)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		for i, n := range d.Nodes {
			if n.ID == "f" && (s[i].Shape.W != c.w || s[i].Shape.H != c.h) {
				t.Errorf("%s: %v x %v, want %v x %v", c.name, s[i].Shape.W, s[i].Shape.H, c.w, c.h)
			}
		}
	}
}
