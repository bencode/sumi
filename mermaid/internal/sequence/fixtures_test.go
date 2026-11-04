package sequence

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// TestFixtures lays out Mermaid's own sequence test diagrams (testdata/e2e,
// from Mermaid v12.3.0): each must lay out, or be refused as not supported
// yet. Those that lay out must keep participants apart, and blocks around
// what is in them.
func TestFixtures(t *testing.T) {
	var ok, unsupported int
	err := filepath.WalkDir("testdata/e2e", func(path string, de fs.DirEntry, err error) error {
		if err != nil || de.IsDir() || !strings.HasSuffix(path, ".mmd") {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		d, err := read(string(src))
		var e *syntax.Error
		if errors.As(err, &e) && e.IsUnsupported() {
			unsupported++
			return nil
		}
		if err != nil {
			t.Errorf("%s: %v", path, err)
			return nil
		}
		ok++
		l := arrange(d, scene.Arial)
		checkApart(t, path, l)
		checkHeld(t, path, l)
		draw(l)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d laid out, %d not supported yet", ok, unsupported)
}

// TestEdges lays out diagrams Mermaid's own do not cover.
func TestEdges(t *testing.T) {
	for _, src := range []string{
		"sequenceDiagram\nloop x\nend",
		"sequenceDiagram\nA->>+B: x\nB->>-A: y",
	} {
		d, err := read(src)
		if err != nil {
			t.Fatalf("%q: %v", src, err)
		}
		l := arrange(d, scene.Arial)
		checkApart(t, src, l)
		checkHeld(t, src, l)
		draw(l)
	}
}

// TestBarTop checks that a bar starts where the message after it does, as
// Mermaid's bounds take the first open bar's top up to it.
func TestBarTop(t *testing.T) {
	d, err := read("sequenceDiagram\nA->>+B: x\nB->>-A: y")
	if err != nil {
		t.Fatal(err)
	}
	var bar *pBar
	var msgs []*pMessage
	for _, p := range arrange(d, scene.Arial).items {
		switch p := p.(type) {
		case *pBar:
			bar = p
		case *pMessage:
			msgs = append(msgs, p)
		}
	}
	if bar == nil || len(msgs) != 2 || bar.box.Y != msgs[1].top {
		t.Errorf("bar %+v, messages %d: the bar should start at the second message's top", bar, len(msgs))
	}
}

// TestCentralBars checks that a bar opened after a central connection
// stacks on the bar the connection opened, as Mermaid's do: Bob's bar
// starts on his lifeline, not half a bar's width left of it.
func TestCentralBars(t *testing.T) {
	d, err := read("sequenceDiagram\nAlice ()->>() Bob: x\nactivate Bob\ndeactivate Bob")
	if err != nil {
		t.Fatal(err)
	}
	l := arrange(d, scene.Arial)
	var bars []*pBar
	for _, p := range l.items {
		if b, ok := p.(*pBar); ok {
			bars = append(bars, b)
		}
	}
	if c := l.e.x[l.e.at["Bob"]]; len(bars) != 1 || bars[0].box.X != c {
		t.Errorf("bars %+v: want one starting at Bob's lifeline, %v", bars, c)
	}
}

func checkApart(t *testing.T, path string, l layout) {
	for i := 1; i < len(l.e.x); i++ {
		if right, left := l.e.x[i-1]+l.e.w[i-1]/2, l.e.x[i]-l.e.w[i]/2; right > left {
			t.Errorf("%s: participants %d and %d overlap", path, i-1, i)
		}
	}
}

// checkHeld checks that each block's frame holds the messages, notes and
// blocks in it, and each box's its participants, at the top and at the
// foot. An activation bar may start in a block and end outside it, so bars
// are left out.
func checkHeld(t *testing.T, path string, l layout) {
	frames := map[string]scene.Rect{}
	for _, p := range l.items {
		switch p := p.(type) {
		case *pBlock:
			frames[p.id] = p.box
		case *pRect:
			frames[p.id] = p.box
		}
	}
	for _, p := range l.items {
		parent := ""
		switch p := p.(type) {
		case *pMessage:
			parent = p.parent
		case *pNote:
			parent = p.parent
		case *pBlock:
			parent = p.parent
		case *pRect:
			parent = p.parent
		}
		if f, ok := frames[parent]; ok && union(f, p.extent()) != f {
			t.Errorf("%s: %v sticks out of %s", path, p.extent(), parent)
		}
	}
	for _, b := range l.boxes {
		for i, p := range l.d.Participants {
			if p.Group != b.Group {
				continue
			}
			for _, foot := range []bool{false, true} {
				y := l.top
				if foot {
					y = l.bottom
				}
				if box := participant(p, i, y, foot, l).Box; union(b.frame, box) != b.frame {
					t.Errorf("%s: %s, foot %v, %v sticks out of %s", path, p.ID, foot, box, b.id)
				}
			}
		}
	}
}
