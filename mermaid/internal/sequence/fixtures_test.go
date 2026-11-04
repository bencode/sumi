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

func checkApart(t *testing.T, path string, l layout) {
	for i := 1; i < len(l.e.x); i++ {
		if right, left := l.e.x[i-1]+l.e.w[i-1]/2, l.e.x[i]-l.e.w[i]/2; right > left {
			t.Errorf("%s: participants %d and %d overlap", path, i-1, i)
		}
	}
}

// checkHeld checks that each block's frame holds the messages, notes and
// blocks in it. An activation bar may start in a block and end outside it,
// so bars are left out.
func checkHeld(t *testing.T, path string, l layout) {
	frames := map[string]scene.Rect{}
	for _, p := range l.items {
		if b, ok := p.(*pBlock); ok {
			frames[b.id] = b.box
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
		}
		if f, ok := frames[parent]; ok && union(f, p.extent()) != f {
			t.Errorf("%s: %v sticks out of %s", path, p.extent(), parent)
		}
	}
}
