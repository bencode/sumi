package gantt

import (
	"math"
	"testing"
	"time"

	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// The places are those of what Mermaid 12.3.0 drew in Chrome, in Shanghai, in Arial.

func drawn(t *testing.T, src string, width float64) *scene.Scene {
	t.Helper()
	zone, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	body, title := syntax.Prepare(src)
	s, err := Layout(body, title, scene.Arial, Env{Now: time.Date(2026, 11, 9, 10, 30, 15, 0, zone), Zone: zone}, width)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// lineReading is the line of a text that reads s, and its font.
func lineReading(t *testing.T, s *scene.Scene, text string) (scene.Line, scene.Font) {
	t.Helper()
	for _, it := range s.Items {
		for _, tx := range it.Texts {
			for _, l := range tx.Lines {
				if l.S == text {
					return l, tx.Font
				}
			}
		}
	}
	t.Fatalf("no text %q", text)
	return scene.Line{}, scene.Font{}
}

// checkAnchor checks that the line reading text has its anchor at x, the part k of its
// width left of it, on baseline.
func checkAnchor(t *testing.T, s *scene.Scene, text string, k, x, baseline float64) {
	t.Helper()
	l, f := lineReading(t, s, text)
	if at := l.X + k*scene.Arial.Measure(l.S, f).Width; math.Abs(at-x) > 0.02 || l.Baseline != baseline {
		t.Errorf("%q anchored at %v on %v, want %v on %v", text, at, l.Baseline, x, baseline)
	}
}

func nearBox(a, b scene.Rect) bool {
	return math.Abs(a.X-b.X) <= 0.02 && math.Abs(a.Y-b.Y) <= 0.02 && math.Abs(a.W-b.W) <= 0.02 && math.Abs(a.H-b.H) <= 0.02
}

func TestATaskTextGoesInItsBarOrBeside(t *testing.T) {
	s := drawn(t, "gantt\ndateFormat YYYY-MM-DD\ntitle T\nexcludes friday\nsection S\n starts :2014-01-15, 2w\n"+
		"section 2024\n中文名字 :2014-03-12, 3d\nends  :2014-03-10, 1d", 400)
	checkAnchor(t, s, "starts", 0.5, 108.5, 63.5) // in the middle of its bar
	checkAnchor(t, s, "ends", 0, 309, 111.5)      // right of its bar
	checkAnchor(t, s, "中文名字", 1, 303, 87.5)       // left of its bar, which ends too near the right
}

// A milestone's text is placed beside the square, which the x moves to the middle of the
// milestone's times, but anchored as beside one at its start, which the class looks at:
// here right of it and ending there. A vert is a thin bar through the rows, its text under
// them in its middle.
func TestAMilestoneTextIsPlacedAndAnchoredApart(t *testing.T) {
	s := drawn(t, "gantt\ndateFormat YYYY-MM-DD\nsection S\nShort :milestone, 2014-02-15, 0d\nA #amp; B :vert, 2014-01-26, 0d\n"+
		"section S\n中文名字 :active, 2014-01-23, 3d", 740)
	var boxes []scene.Rect
	for _, it := range s.Items {
		if it.Kind == scene.Node {
			boxes = append(boxes, it.Box)
		}
	}
	d := 16 * math.Sqrt2 // the square of 20, scaled by 0.8 and turned 45°
	want := []scene.Rect{{X: 75, Y: 74, W: 77, H: 20}, {X: 665 - d/2, Y: 60 - d/2, W: d, H: d}, {X: 152, Y: 35, W: 1.6, H: 88}}
	if len(boxes) != len(want) {
		t.Fatalf("bars %v, want %v", boxes, want)
	}
	for i := range want {
		if !nearBox(boxes[i], want[i]) {
			t.Errorf("bar %d is %v, want %v", i, boxes[i], want[i])
		}
	}
	checkAnchor(t, s, "Short", 1, 680, 63.5)
	if _, f := lineReading(t, s, "Short"); !f.Italic {
		t.Error("a milestone's text is upright")
	}
	checkAnchor(t, s, "A & B", 0.5, 152, 143)
	if _, f := lineReading(t, s, "A & B"); f.Size != 15 {
		t.Errorf("a vert's text is %v, want 15", f.Size)
	}
}

// The renderer measures a text before Mermaid puts its entity codes back: A ﬂ°amp¶ß B does
// not fit the bar that A & B would.
func TestEntityCodesAreMeasuredAsPlaceholders(t *testing.T) {
	s := drawn(t, "gantt\ndateFormat YYYY-MM-DD\nsection Section one\nA #amp; B :2014-01-01, 3d\n"+
		"Task one :a1, 2014-01-01, 2014-03-01\nTask one :crit, 2014-02-28, 1M\na-b :crit, 2014-01-01, 3d", 1200)
	checkAnchor(t, s, "A & B", 0, 117, 63.5)
}

// The titles go by the keys of an object, a name that is an array index first, while the
// rows go as written; each line's middle is where its baseline would be.
func TestSectionTitlesGoByTheirKeys(t *testing.T) {
	s := drawn(t, "gantt\ndateFormat YYYY-MM-DD\nsection Z\nT1 :2014-01-01, 2d\nsection a<br/>b<BR />c\nT2 :2014-01-02, 2d\n"+
		"T3 :2014-01-03, 1d\nsection 7\nT4 :2014-01-01, 1d", 740)
	for _, w := range []struct {
		s string
		y float64
	}{{"7", 66}, {"Z", 90}, {"a", 115}, {"b", 126}, {"c", 137}} {
		checkAnchor(t, s, w.s, 0, 10, w.y)
	}
}

// A run of excluded days still open at the latest end, January 18 here, is not drawn.
func TestTheLastRunOfExcludedDaysIsNotDrawn(t *testing.T) {
	s := drawn(t, "gantt\ndateFormat YYYY-MM-DD\nexcludes weekends\nsection S\nA :2014-01-03, 2014-01-18", 2000)
	var runs []scene.Rect
	for _, it := range s.Items {
		if it.Kind == scene.Mark && it.Box.Y == gridLineStartPadding && len(it.Paths[0].Segs) > 2 {
			runs = append(runs, it.Box)
		}
	}
	want := []scene.Rect{{X: 198, Y: 35, W: 247, H: 39}, {X: 1062, Y: 35, W: 246, H: 39}}
	if len(runs) != len(want) || !nearBox(runs[0], want[0]) || !nearBox(runs[1], want[1]) {
		t.Errorf("excluded days %v, want %v", runs, want)
	}
}
