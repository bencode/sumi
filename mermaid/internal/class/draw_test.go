package class

import (
	"math"
	"slices"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

func drawn(t *testing.T, src string) *scene.Scene {
	t.Helper()
	body, title := syntax.Prepare(src)
	s, err := Layout(body, title, scene.Arial)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func node(t *testing.T, s *scene.Scene, id string) scene.Item {
	t.Helper()
	i := slices.IndexFunc(s.Items, func(it scene.Item) bool { return it.Kind == scene.Node && it.ID == id })
	if i < 0 {
		t.Fatalf("no node %q in %+v", id, s.Items)
	}
	return s.Items[i]
}

// An author's stroke reaches the lines across a class as well as its outline, as Chrome
// draws them: the stylesheet's stroke-width beats the 1.3 the shape sets.
func TestAStyleReachesAClassDividers(t *testing.T) {
	it := node(t, drawn(t, "classDiagram\nclass A\nstyle A stroke:#c00,stroke-width:3px"), "A")
	if len(it.Paths) != 3 {
		t.Fatalf("paths %+v", it.Paths)
	}
	for _, p := range it.Paths {
		if p.Stroke.Width != 3 || p.Stroke.Paint != scene.Fixed(scene.Color{R: 0xcc, A: 255}) {
			t.Errorf("stroke %+v", p.Stroke)
		}
	}
}

// A static member is underlined from where its text starts to where it ends, one pixel
// under its baseline.
func TestAStaticMemberIsUnderlined(t *testing.T) {
	it := node(t, drawn(t, "classDiagram\nclass A {\n +x$\n +y\n}"), "A")
	if len(it.Paths) != 4 {
		t.Fatalf("paths %+v", it.Paths)
	}
	bar := it.Paths[3].Segs[0].Pts[0]
	var baseline float64
	for _, x := range it.Texts {
		for _, l := range x.Lines {
			if l.S == "+x" {
				baseline = l.Baseline
			}
		}
	}
	if math.Abs(bar.Y-baseline-underlineGap) > 1e-9 {
		t.Errorf("the line starts at %v, the baseline is %v", bar.Y, baseline)
	}
}

func TestATextAtAnEdgeEndIs11Px(t *testing.T) {
	s := drawn(t, "classDiagram\nA \"1\" --> B")
	i := slices.IndexFunc(s.Items, func(it scene.Item) bool { return it.Kind == scene.Label })
	if got := s.Items[i].Texts[0].Font.Size; got != endTextSize {
		t.Errorf("size %v", got)
	}
	if len(s.Items[i].Paths) != 0 {
		t.Errorf("a patch behind it: %+v", s.Items[i].Paths)
	}
}
