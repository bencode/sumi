package class

import (
	"errors"
	"math"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/elk"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

func placeSource(t *testing.T, source string) (Placement, error) {
	t.Helper()
	d, err := Read(source)
	if err != nil {
		t.Fatal(err)
	}
	return Place(d, scene.Arial)
}

// The centers come from Mermaid 12.3.0 in Chrome. ELK puts both texts on the same
// side of the line; the one at the start is mirrored to the right of the direction
// of travel, and the one at the end, with a marker to clear, stays.
func TestTextsAtEdgeEnds(t *testing.T) {
	p, err := placeSource(t, "classDiagram\nA \"1\" --> \"many\" B")
	if err != nil {
		t.Fatal(err)
	}
	near := func(got *elk.Vec, x, y float64) bool {
		return got != nil && math.Abs(got.X-x) < 0.02 && math.Abs(got.Y-y) < 0.02
	}
	e := p.R.Edges[0]
	if !near(e.Start, 21, 105.25) || !near(e.End, 53.515625, 167.75) {
		t.Errorf("start %v end %v", e.Start, e.End)
	}
}

func TestClassNamedAsANamespaceIsRefused(t *testing.T) {
	_, err := placeSource(t, "classDiagram\nclass N\nnamespace N {\n class X\n}")
	var e *syntax.Error
	if !errors.As(err, &e) || !e.IsUnsupported() {
		t.Errorf("got %v", err)
	}
}
