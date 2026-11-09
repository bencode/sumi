package er

import (
	"math"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

func sized(t *testing.T, src string) ([]Sized, error) {
	t.Helper()
	body, _ := syntax.Prepare("erDiagram\n" + src)
	d, err := Read(body)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return Size(d, scene.Arial)
}

func near(a, b float64) bool { return math.Abs(a-b) < 0.05 }

// The sizes are what Chrome measured of Mermaid 12.3.0's erBox.
func TestEntitySizeAsChromeMeasuredIt(t *testing.T) {
	for _, c := range []struct {
		name, src string
		w, h      float64
		columns   []float64
	}{
		{"the least width", "A", 100, 81, nil},
		{"under the least, past its estimate", "aaaaaaa", 94.516, 81, nil},
		{"wider than the least", "AVeryLongEntityNameThatIsWide", 252.438, 81, nil},
		{"a label that wraps", `"aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa"`, 160, 165, nil},
		{"two columns", "A {\n string name\n}", 119.266, 79.5, []float64{-0.398}},
		{"four columns", "A {\n string name PK \"the name\"\n int age FK,UK\n}", 268.875, 119.25, []float64{-75.203, -15.172, 51.062}},
		{"a name wider than the columns", "AVeryLongEntityNameThatIsMuchWiderThanTheRows {\n a b\n}", 392.375, 79.5, []float64{0}},
		{"a comment of two lines", "A {\n a b \"x<br>y\"\n}", 97.594, 100.5, nil},
		{"a generic loses its style", "A {\n a b \"c **d~e~**\"\n}", 133.422, 79.5, nil},
	} {
		s, err := sized(t, c.src)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !near(s[0].Shape.W, c.w) || !near(s[0].Shape.H, c.h) {
			t.Errorf("%s: %v x %v, want %v x %v", c.name, s[0].Shape.W, s[0].Shape.H, c.w, c.h)
		}
		for i, x := range c.columns {
			if !near(s[0].Table.Columns[i], x) {
				t.Errorf("%s: columns %v, want %v", c.name, s[0].Table.Columns, c.columns)
			}
		}
	}
}

func TestTableCellsAsChromeMeasuredThem(t *testing.T) {
	s, err := sized(t, "A {\n string name PK \"the name\"\n int age FK,UK\n}")
	if err != nil {
		t.Fatal(err)
	}
	tb := s[0].Table
	for _, c := range []struct {
		what string
		got  Placed
		x, y float64
	}{
		{"name", tb.Name, -4.672, -50.25},
		{"type", tb.Rows[0].Type, -121.938, -10.5},
		{"keys", tb.Rows[1].Keys, -2.672, 29.25},
		{"comment", tb.Rows[0].Comment, 63.563, -10.5},
	} {
		if !near(c.got.At.X, c.x) || !near(c.got.At.Y, c.y) {
			t.Errorf("%s at %v, want %v %v", c.what, c.got.At, c.x, c.y)
		}
	}
}

func TestEntityTextErrors(t *testing.T) {
	for _, c := range []struct {
		name, src   string
		unsupported bool
	}{
		{"HTML in a comment", "A {\n a b \"<i>x</i>\"\n}", true},
		{"a blank comment, which Mermaid throws on", "A {\n a b \" \"\n}", false},
		{"a blank line in a label", "A[\"x<br> \"]", false},
	} {
		_, err := sized(t, c.src)
		e, ok := err.(*syntax.Error)
		if !ok || e.IsUnsupported() != c.unsupported {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}
