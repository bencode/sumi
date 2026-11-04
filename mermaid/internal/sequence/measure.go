package sequence

import (
	"math"
	"regexp"
	"strings"

	"github.com/bencode/sumi/scene"
)

// A textBlock is text measured the way Mermaid measures it
// (calculateTextDimensions, utils.ts): split at <br>, the widest line and the
// height of a line each rounded to whole pixels.
type textBlock struct {
	lines     []string
	w, lh     float64
	font      scene.Font
	asc, desc float64
}

var breaks = regexp.MustCompile(`(?i)<br\s*/?>`)

func measure(m scene.Measurer, s string, size float64) textBlock {
	tb := textBlock{font: scene.Font{Size: size}}
	if s == "" {
		return tb
	}
	for _, l := range breaks.Split(s, -1) {
		l = strings.Join(strings.Fields(l), " ") // spaces as SVG sets them
		met := m.Measure(l, tb.font)
		tb.lines = append(tb.lines, l)
		tb.w = max(tb.w, math.Round(met.Width))
		tb.asc, tb.desc = met.Ascent, met.Descent
	}
	tb.lh = math.Round(tb.asc + tb.desc)
	return tb
}

func (tb textBlock) h() float64 { return tb.lh * float64(len(tb.lines)) }

// middle is the baseline that puts a line's middle at y, as SVG's
// dominant-baseline: middle does.
func (tb textBlock) middle(y float64) float64 { return y + (tb.asc-tb.desc)/2 }

// set lays out tb's lines, each centered on x, or starting at x when left,
// line i on baseline base(i). It returns the text and the box it fills.
func (tb textBlock) set(m scene.Measurer, x float64, left bool, base func(i int) float64) (scene.Text, scene.Rect) {
	t := scene.Text{Font: tb.font, Paint: scene.Use(scene.TextColor)}
	var box scene.Rect
	for i, l := range tb.lines {
		w := m.Measure(l, tb.font).Width
		x0 := x - w/2
		if left {
			x0 = x
		}
		t.Lines = append(t.Lines, scene.Line{S: l, X: x0, Baseline: base(i)})
		box = union(box, scene.Rect{X: x0, Y: base(i) - tb.asc, W: w, H: tb.asc + tb.desc})
	}
	return t, box
}

// widths are the participants' widths: the least width, or wide enough for
// the label and padding (calculateActorMargins, sequenceRenderer.ts).
func widths(ps []*Participant, m scene.Measurer, c config) []float64 {
	w := make([]float64, len(ps))
	for i, p := range ps {
		w[i] = max(c.width, measure(m, p.Label, c.actorFont).w+2*c.wrapPadding)
	}
	return w
}

// wrapLabel breaks label into lines no wider than maxWidth, joined by <br/>,
// as Mermaid's wrapLabel (utils.ts) does: at spaces, and a word too wide for
// a line of its own is broken into hyphenated pieces. A label with a <br>
// in it is left as it is.
func wrapLabel(m scene.Measurer, label string, maxWidth, size float64) string {
	if label == "" || breaks.MatchString(label) {
		return label
	}
	width := func(s string) float64 { return measure(m, s, size).w }
	var lines []string
	next := ""
	for _, word := range strings.Fields(label) {
		wl, nl := width(word+" "), width(next)
		switch {
		case wl > maxWidth:
			pieces, rest := breakWord(word, maxWidth, width)
			lines = append(append(lines, next), pieces...)
			next = rest
		case nl+wl >= maxWidth:
			lines = append(lines, next)
			next = word
		case next == "":
			next = word
		default:
			next += " " + word
		}
	}
	lines = append(lines, next)
	kept := lines[:0]
	for _, l := range lines {
		if l != "" {
			kept = append(kept, l)
		}
	}
	return strings.Join(kept, "<br/>")
}

// breakWord is Mermaid's breakString: word cut, a character at a time, into
// pieces as wide as maxWidth, each but the last ending in a hyphen, and
// what is left over.
func breakWord(word string, maxWidth float64, width func(string) float64) ([]string, string) {
	rs := []rune(word)
	var pieces []string
	cur := ""
	for i, r := range rs {
		next := cur + string(r)
		if width(next) < maxWidth {
			cur = next
			continue
		}
		if i < len(rs)-1 {
			next += "-"
		}
		pieces = append(pieces, next)
		cur = ""
	}
	return pieces, cur
}

// union is the smallest box holding a and b; an empty box holds nothing.
func union(a, b scene.Rect) scene.Rect {
	if a == (scene.Rect{}) {
		return b
	}
	if b == (scene.Rect{}) {
		return a
	}
	x0, y0 := min(a.X, b.X), min(a.Y, b.Y)
	x1, y1 := max(a.X+a.W, b.X+b.W), max(a.Y+a.H, b.Y+b.H)
	return scene.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// between is the box between two corners, in either order.
func between(x0, y0, x1, y1 float64) scene.Rect {
	return scene.Rect{X: min(x0, x1), Y: min(y0, y1), W: math.Abs(x1 - x0), H: math.Abs(y1 - y0)}
}
