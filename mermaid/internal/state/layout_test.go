package state

import (
	"slices"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/graphlayout"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

func placed(t *testing.T, src string) (Placed, error) {
	t.Helper()
	body, _ := syntax.Prepare(src)
	d, err := Read(body)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return Place(d, scene.Arial)
}

func TestPlaceRefusesATransitionAtRoot(t *testing.T) {
	if _, err := placed(t, "stateDiagram-v2\nroot --> A"); err == nil {
		t.Error("got no error")
	}
}

// A note group's label is the note's text, which the note inside it wraps: on one line it
// sticks out of the frame, which moves the frame over and leaves it no wider.
func TestNoteTextSticksOutOfItsGroup(t *testing.T) {
	group := func(text string) graphlayout.PlacedNode {
		p, err := placed(t, "stateDiagram-v2\nA\nnote right of A : "+text)
		if err != nil {
			t.Fatal(err)
		}
		i := slices.IndexFunc(p.R.Nodes, func(n graphlayout.PlacedNode) bool { return n.ID == "A----parent" })
		if i < 0 {
			t.Fatalf("no note group in %+v", p.R.Nodes)
		}
		return p.R.Nodes[i]
	}
	short, long := group("a"), group("a note long enough that on one line it would be far wider than the note is")
	if long.Width != short.Width || long.X <= short.X {
		t.Errorf("short %+v, long %+v", short, long)
	}
}
