package er_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/bencode/sumi/mermaid"
	"github.com/bencode/sumi/mermaid/internal/er"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

func drawn(t *testing.T, src string) *scene.Scene {
	t.Helper()
	body, title := syntax.Prepare("erDiagram\n" + src)
	s, err := er.Layout(body, title, scene.Arial)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func item(t *testing.T, s *scene.Scene, k scene.Kind) scene.Item {
	t.Helper()
	i := slices.IndexFunc(s.Items, func(it scene.Item) bool { return it.Kind == k })
	if i < 0 {
		t.Fatalf("no item of kind %v in %+v", k, s.Items)
	}
	return s.Items[i]
}

func near(a, b float64) bool { return math.Abs(a-b) < 0.02 }

// markerPoints are the points of a line's markers, and the center of a filled circle.
func markerPoints(paths []scene.Path) [][2]float64 {
	var out [][2]float64
	for _, p := range paths {
		var pts []scene.Point
		for _, s := range p.Segs {
			n := map[scene.Op]int{scene.MoveTo: 1, scene.LineTo: 1, scene.QuadTo: 2, scene.CubeTo: 3}[s.Op]
			pts = append(pts, s.Pts[:n]...)
		}
		if p.Fill == (scene.Paint{}) {
			for _, q := range pts {
				out = append(out, [2]float64{q.X, q.Y})
			}
			continue
		}
		xs := make([]float64, len(pts))
		ys := make([]float64, len(pts))
		for i, q := range pts {
			xs[i], ys[i] = q.X, q.Y
		}
		out = append(out, [2]float64{(slices.Min(xs) + slices.Max(xs)) / 2, (slices.Min(ys) + slices.Max(ys)) / 2})
	}
	return out
}

// The points are where Chrome drew Mermaid 12.3.0's markers on a line going down.
func TestMarkersAreWhereChromeDrawsThem(t *testing.T) {
	lensAbove := [][2]float64{{58, 71}, {76, 89}, {58, 107}, {40, 89}, {58, 71}}
	lensBelow := [][2]float64{{58, 172}, {76, 190}, {58, 208}, {40, 190}, {58, 172}}
	for _, c := range []struct {
		op   string
		want [][2]float64
	}{
		{"||--||", [][2]float64{{67, 98}, {49, 98}, {67, 104}, {49, 104}, {67, 175}, {49, 175}, {67, 181}, {49, 181}}},
		{"|o--o|", [][2]float64{{58, 110}, {67, 98}, {49, 98}, {58, 169}, {67, 181}, {49, 181}}},
		{"}|--|{", slices.Concat(lensAbove, [][2]float64{{67, 113}, {49, 113}, {67, 166}, {49, 166}}, lensBelow)},
		{"}o--o{", slices.Concat([][2]float64{{58, 116.5}}, lensAbove, [][2]float64{{58, 162}}, lensBelow)},
	} {
		got := markerPoints(item(t, drawn(t, "direction TB\nAlpha "+c.op+" Beta : x"), scene.Edge).Paths[1:])
		if len(got) != len(c.want) {
			t.Errorf("%s: %v, want %v", c.op, got, c.want)
			continue
		}
		for _, w := range c.want {
			if !slices.ContainsFunc(got, func(g [2]float64) bool { return near(g[0], w[0]) && near(g[1], w[1]) }) {
				t.Errorf("%s: no point at %v in %v", c.op, w, got)
			}
		}
	}
}

func TestATableIsSetFromTheLeft(t *testing.T) {
	var xs []float64
	for _, x := range item(t, drawn(t, "A {\n int n \"a<br>b<br>c\"\n}"), scene.Node).Texts {
		for _, l := range x.Lines {
			if l.S == "a" || l.S == "b" || l.S == "c" {
				xs = append(xs, l.X)
			}
		}
	}
	if len(xs) != 3 || !near(xs[0], 93.09375) || !near(xs[1], 93.09375) || !near(xs[2], 93.09375) {
		t.Errorf("the comment's lines start at %v", xs)
	}
}

func TestAStyleReachesAllOfATable(t *testing.T) {
	it := item(t, drawn(t, "A {\n int n PK \"c\"\n string m\n}\nstyle A fill:#def,stroke:#c00,stroke-width:3px,color:#900"), scene.Node)
	if len(it.Paths) != 7 { // the outline, two rows, the line under the name, three columns
		t.Fatalf("paths %+v", it.Paths)
	}
	red := scene.Fixed(scene.Color{R: 0xcc, A: 255})
	for _, p := range it.Paths {
		if p.Stroke.Width != 3 || p.Stroke.Paint != red {
			t.Errorf("stroke %+v", p.Stroke)
		}
	}
	for _, x := range it.Texts {
		if x.Paint != scene.Fixed(scene.Color{R: 0x99, A: 255}) {
			t.Errorf("text %+v", x.Paint)
		}
	}
}

func TestASubgraphsStyleAndTitle(t *testing.T) {
	s := drawn(t, "subgraph S[\"A subgraph title long enough that it has to wrap onto a second line\"]\n A\nend\nstyle S stroke:#c00,color:#900")
	if !near(s.Width, 172) { // the title is set again in 148, and the frame widens to 156
		t.Errorf("width %v", s.Width)
	}
	it := item(t, s, scene.Group)
	if p := it.Paths[0]; p.Stroke.Width != 1 || p.Stroke.Paint != scene.Fixed(scene.Color{R: 0xcc, A: 255}) {
		t.Errorf("frame %+v", p.Stroke)
	}
	if len(it.Texts) != 1 || len(it.Texts[0].Lines) != 3 || it.Texts[0].Paint != scene.Fixed(scene.Color{R: 0x99, A: 255}) {
		t.Errorf("title %+v", it.Texts)
	}
}

func TestAFontStyleIsRefused(t *testing.T) {
	body, title := syntax.Prepare("erDiagram\nA\nstyle A font-size:20px")
	_, err := er.Layout(body, title, scene.Arial)
	var e *syntax.Error
	if !errors.As(err, &e) || !e.IsUnsupported() {
		t.Errorf("%v", err)
	}
}

func TestERDiagramsAreRegistered(t *testing.T) {
	if _, err := mermaid.Layout("erDiagram\nA ||--o{ B : x", mermaid.Options{}); err != nil {
		t.Error(err)
	}
}
