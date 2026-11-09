package class

import (
	"math"
	"strings"
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

func near(a, b float64) bool { return math.Abs(a-b) < 0.05 }

// The sizes are what Chrome measured of Mermaid 12.3.0's classBox. The box of a class
// with no members keeps room for both compartments.
func TestClassBoxAsChromeMeasuredIt(t *testing.T) {
	for _, c := range []struct {
		name, src string
		w, h      float64
		dividers  [2]float64
		titleTop  float64
	}{
		{"no members", "classDiagram\nclass A", 34.125, 81, [2]float64{4.5, 22.5}, -27},
		{"an attribute", "classDiagram\nclass A {\n +int x\n}", 62.922, 114, [2]float64{-12, 33}, -43.5},
		{"a method", "classDiagram\nclass A {\n go()\n}", 53.969, 120, [2]float64{-15, 9}, -46.5},
		{"both", "classDiagram\nclass A {\n +int x\n -String y$\n go() int*\n}", 81.188, 156, [2]float64{-33, 33}, -64.5},
	} {
		_, s, err := sized(t, c.src)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		got := s[0]
		if !near(got.Shape.W, c.w) || !near(got.Shape.H, c.h) {
			t.Errorf("%s: size %v x %v, want %v x %v", c.name, got.Shape.W, got.Shape.H, c.w, c.h)
		}
		if b := got.Class; !near(b.Dividers[0], c.dividers[0]) || !near(b.Dividers[1], c.dividers[1]) || !near(b.Title.At.Y, c.titleTop) {
			t.Errorf("%s: dividers %v, title top %v, want %v, %v", c.name, b.Dividers, b.Title.At.Y, c.dividers, c.titleTop)
		}
	}
}

// Markdown does not see a member's visibility, so a [ after it is not a link beside an
// escape.
func TestVisibilityIsTextBesideABracket(t *testing.T) {
	_, s, err := sized(t, "classDiagram\nA : +int[] ids")
	if err != nil {
		t.Fatal(err)
	}
	if line := s[0].Class.Members[0].Box.Lines[0]; line[0].Text != "+int[] ids" {
		t.Errorf("got %+v", line)
	}
}

func TestLabelBesideAGenericType(t *testing.T) {
	_, s, err := sized(t, "classDiagram\nclass B~T~[\"x\"]")
	if err != nil {
		t.Fatal(err)
	}
	if line := s[0].Class.Title.Box.Lines[0]; len(line) != 1 || line[0].Text != "x" || !line[0].Bold {
		t.Errorf("got %+v", line)
	}
	if _, _, err := sized(t, "classDiagram\nclass B~1~[\"x\"]"); err == nil || !err.(*syntax.Error).IsUnsupported() {
		t.Errorf("a type that cannot start a tag: got %v", err)
	}
}

func TestClassifierStylesTheMember(t *testing.T) {
	_, s, err := sized(t, "classDiagram\nA : x*\nA : y$")
	if err != nil {
		t.Fatal(err)
	}
	ms := s[0].Class.Members
	if !ms[0].Box.Lines[0][0].Italic || ms[1].Box.Lines[0][0].Italic {
		t.Errorf("got %+v, %+v", ms[0].Box.Lines, ms[1].Box.Lines)
	}
}

// getData's white-space: nowrap keeps a note on one line, and an interface has only
// flowchart.wrappingWidth.
func TestNoteDoesNotWrapAndAnInterfaceDoes(t *testing.T) {
	long := strings.Repeat("aaaa-", 60) + "a"
	_, s, err := sized(t, "classDiagram\nnote \""+strings.ReplaceAll(long, "-", " ")+"\"\n"+long+" ()-- B")
	if err != nil {
		t.Fatal(err)
	}
	if s[1].Shape.H != 33 {
		t.Errorf("note: height %v", s[1].Shape.H)
	}
	if !near(s[2].Label.W, 120) || s[2].Label.H <= 21 {
		t.Errorf("interface: %v x %v", s[2].Label.W, s[2].Label.H)
	}
}

func TestNoteAndInterfacePadding(t *testing.T) {
	_, s, err := sized(t, "classDiagram\nnote \"t\"\na ()-- B")
	if err != nil {
		t.Fatal(err)
	}
	if !near(s[1].Shape.W, 15.891) || !near(s[1].Shape.H, 33) {
		t.Errorf("note: %v x %v", s[1].Shape.W, s[1].Shape.H)
	}
	if !near(s[2].Shape.W, 39.797) || !near(s[2].Shape.H, 45) {
		t.Errorf("interface: %v x %v", s[2].Shape.W, s[2].Shape.H)
	}
}
