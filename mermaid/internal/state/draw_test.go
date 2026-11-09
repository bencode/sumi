package state

import (
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

func item(t *testing.T, s *scene.Scene, kind scene.Kind, id string) scene.Item {
	t.Helper()
	i := slices.IndexFunc(s.Items, func(it scene.Item) bool { return it.Kind == kind && it.ID == id })
	if i < 0 {
		t.Fatalf("no %v %q in %+v", kind, id, s.Items)
	}
	return s.Items[i]
}

// A note's line is dotted, as neo draws a dashed pattern, and has no arrow to fill.
func TestANoteLineIsDashedWithNoArrow(t *testing.T) {
	s := drawn(t, "stateDiagram-v2\nA\nnote right of A : t")
	i := slices.IndexFunc(s.Items, func(it scene.Item) bool { return it.Kind == scene.Edge })
	for _, p := range s.Items[i].Paths {
		if p.Fill != (scene.Paint{}) {
			t.Errorf("a filled path, an arrow: %+v", p)
		}
	}
	if len(s.Items[i].Paths[0].Stroke.Dash) == 0 {
		t.Errorf("not dashed: %+v", s.Items[i].Paths[0].Stroke)
	}
}

// A note group's frame shows nothing, and the diagram still has room for it.
func TestANoteGroupDrawsNothingButTakesRoom(t *testing.T) {
	s := drawn(t, "stateDiagram-v2\nA\nnote right of A : t")
	g := item(t, s, scene.Group, "A----parent")
	if len(g.Paths) != 0 {
		t.Errorf("paths %+v", g.Paths)
	}
	if got := g.Box.X + g.Box.W + 8; got != s.Width {
		t.Errorf("the frame ends at %v, the scene at %v", got, s.Width)
	}
}

// A composite is an outer frame and an inner one below the title; one with nothing in it
// has no height, and SVG draws neither.
func TestACompositeHasTwoFrames(t *testing.T) {
	g := item(t, drawn(t, "stateDiagram-v2\nstate X {\nA\n}"), scene.Group, "X")
	if len(g.Paths) != 2 || g.Paths[1].Segs[0].Pts[0].Y <= g.Paths[0].Segs[0].Pts[0].Y {
		t.Errorf("paths %+v", g.Paths)
	}
	if g := item(t, drawn(t, "stateDiagram-v2\nstate X {\n}"), scene.Group, "X"); len(g.Paths) != 0 {
		t.Errorf("an empty composite draws %+v", g.Paths)
	}
}
