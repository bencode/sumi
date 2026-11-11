package pie

import (
	"math"
	"slices"
	"strconv"

	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// What pieRenderer and the default theme draw a pie chart with, in pixels.
const (
	height      = 450
	margin      = 40
	radius      = height/2 - margin
	legendBox   = 18 // LEGEND_RECT_SIZE
	legendGap   = 4  // LEGEND_SPACING
	legendStep  = legendBox + legendGap
	legendLeft  = 12 * legendBox
	labelRadius = 0.75 * radius // textPosition
	strokeWidth = 2             // pieStrokeWidth and pieOuterStrokeWidth
	highlighted = 1.05          // the scale of the .highlighted class
	titleSize   = 25
	textSize    = 17 // pieSectionTextSize and pieLegendTextSize
)

// Layout lays out a diagram's body, prepared by syntax.Prepare, as pieRenderer draws it.
// title is the front matter's, which a title statement takes the place of.
func Layout(body, title string, m scene.Measurer) (*scene.Scene, error) {
	d, err := Read(body)
	if err != nil {
		return nil, err
	}
	measured := title
	if d.Title != "" {
		title, measured = syntax.Decode(d.Title), syntax.Placeholders(d.Title)
	}
	title, measured = label.Collapsed([]string{title})[0], label.Collapsed([]string{measured})[0]
	titleW := m.Measure(measured, scene.Font{Size: titleSize}).Width
	// The pie's center, where the drawing's left edge is at 0: the title, centered on it,
	// may reach past the left of the pie's box.
	left := min(0, height/2-titleW/2)
	c := scene.Point{X: height/2 - left, Y: height / 2}
	keys, longest := legend(d, c, m)
	right := max(height+margin+legendStep+longest, height/2+titleW/2)
	ss := slicesOf(d.Sections)
	items := slices.Concat([]scene.Item{outerCircle(c)}, pieces(c, ss), percents(c, ss, m),
		drawTitle(c, title, m), keys)
	return &scene.Scene{Width: right - left, Height: height, Items: items}, nil
}

// A slice is a section drawn in the pie: its angles, clockwise from 12 o'clock in radians,
// its color's place among all the sections, and its share of their total, from 0 to 1.
type slice struct {
	Section
	a0, a1 float64
	color  int
	share  float64
}

// slicesOf are the sections createPieArcs keeps, a share of the total of all at least 1%,
// which none is when the total is 0 or +Inf; d3's pie shares the circle among them alone,
// in the order written.
func slicesOf(sections []Section) []slice {
	total := 0.0
	for _, s := range sections {
		total += s.Value
	}
	var out []slice
	kept := 0.0
	for i, s := range sections {
		if share := s.Value / total; share*100 >= 1 {
			out = append(out, slice{Section: s, color: i, share: share})
			kept += s.Value
		}
	}
	a := 0.0
	for i := range out {
		out[i].a0 = a
		a += out[i].Value * (2 * math.Pi / kept)
		out[i].a1 = a
	}
	return out
}

func outerCircle(c scene.Point) scene.Item {
	r := radius + strokeWidth/2.0
	return mark(scene.Path{Segs: scene.Ellipse(c, r, r), Stroke: scene.Stroke{Paint: scene.Use(scene.PieOuterStroke), Width: strokeWidth}},
		scene.Rect{X: c.X - r, Y: c.Y - r, W: 2 * r, H: 2 * r})
}

// pieces are the slices as d3's arc draws them: the arc and two lines to the center, or a
// whole circle for a slice that goes all the way round. The slice of the label pieConfig's
// highlightSlice names, which is "" when no config sets it, is the bigger by the scale of
// its class, its stroke too.
func pieces(c scene.Point, ss []slice) []scene.Item {
	out := make([]scene.Item, len(ss))
	for i, s := range ss {
		r, w := float64(radius), float64(strokeWidth)
		if s.Label == "" {
			r, w = r*highlighted, w*highlighted
		}
		p := &scene.Path{Fill: scene.Use(fill(s.color)), Stroke: scene.Stroke{Paint: scene.Use(scene.PieStroke), Width: w}}
		whole := s.a1-s.a0 > 2*math.Pi-1e-12
		start := at(c, r, s.a0)
		p.MoveTo(start.X, start.Y)
		arc(p, c, r, s.a0, s.a1)
		if !whole {
			p.LineTo(c.X, c.Y)
		}
		p.Close()
		out[i] = mark(*p, sectorBox(c, r, s.a0, s.a1, whole))
	}
	return out
}

// arc adds to p the arc of the circle about c, r wide, clockwise from a0 to a1, in cubic
// curves of a quarter turn at most.
func arc(p *scene.Path, c scene.Point, r, a0, a1 float64) {
	n := math.Ceil((a1 - a0) / (math.Pi / 2))
	step := (a1 - a0) / n
	h := 4.0 / 3 * math.Tan(step/4) * r
	for k := range int(n) {
		a, b := a0+float64(k)*step, a0+float64(k+1)*step
		p0, p3 := at(c, r, a), at(c, r, b)
		p.CubeTo(p0.X+h*math.Cos(a), p0.Y+h*math.Sin(a), p3.X-h*math.Cos(b), p3.Y-h*math.Sin(b), p3.X, p3.Y)
	}
}

// at is the point of the circle about c, r wide, at the angle a clockwise from 12 o'clock.
func at(c scene.Point, r, a float64) scene.Point {
	return scene.Point{X: c.X + r*math.Sin(a), Y: c.Y - r*math.Cos(a)}
}

// sectorBox is the box of the sector from a0 to a1: its center, its ends, and the points
// due up, right, down or left that lie between them.
func sectorBox(c scene.Point, r, a0, a1 float64, whole bool) scene.Rect {
	if whole {
		return scene.Rect{X: c.X - r, Y: c.Y - r, W: 2 * r, H: 2 * r}
	}
	pts := []scene.Point{c, at(c, r, a0), at(c, r, a1)}
	for q := math.Ceil(a0 / (math.Pi / 2)); q*math.Pi/2 < a1; q++ {
		pts = append(pts, at(c, r, q*math.Pi/2))
	}
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, p := range pts {
		x0, y0, x1, y1 = min(x0, p.X), min(y0, p.Y), max(x1, p.X), max(y1, p.Y)
	}
	return scene.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// percents are the slices' shares, rounded, centered where the middle of each slice's
// angle meets the circle textPosition of the radius wide, their baselines there.
func percents(c scene.Point, ss []slice, m scene.Measurer) []scene.Item {
	f := scene.Font{Size: textSize}
	out := make([]scene.Item, len(ss))
	for i, s := range ss {
		p := at(c, labelRadius, (s.a0+s.a1)/2)
		out[i] = text(strconv.FormatFloat(math.Round(s.share*100), 'f', -1, 64)+"%", f, m, p.X, p.Y, scene.Center, scene.Use(scene.PieText))
	}
	return out
}

func drawTitle(c scene.Point, title string, m scene.Measurer) []scene.Item {
	if title == "" {
		return nil
	}
	return []scene.Item{text(title, scene.Font{Size: titleSize}, m, c.X, c.Y-(height-50)/2, scene.Center, scene.Use(scene.TextColor))}
}

// legend is a square and a label for each section, the pie's or not, in a column to the
// right of the pie, about its middle; and the width of the widest label, as Math.max gives
// it, -Inf for none. A label measures its entity codes as Mermaid's placeholders.
func legend(d *Diagram, c scene.Point, m scene.Measurer) ([]scene.Item, float64) {
	f := scene.Font{Size: textSize}
	var out []scene.Item
	longest := math.Inf(-1)
	n := len(d.Sections)
	for i, s := range d.Sections {
		x, y := c.X+legendLeft, c.Y+float64(i*legendStep)-float64(n*legendStep)/2
		box := scene.Rect{X: x, Y: y, W: legendBox, H: legendBox}
		paint := scene.Use(fill(i))
		out = append(out, mark(scene.Path{Segs: scene.RoundRect(box, 0), Fill: paint, Stroke: scene.Stroke{Paint: paint, Width: 1}}, box))
		shown, measured := syntax.Decode(s.Label), syntax.Placeholders(s.Label)
		if d.ShowData {
			v := value(s.Value)
			shown, measured = shown+" ["+v+"]", measured+" ["+v+"]"
		}
		shown, measured = label.Collapsed([]string{shown})[0], label.Collapsed([]string{measured})[0]
		longest = max(longest, m.Measure(measured, f).Width)
		if shown != "" {
			out = append(out, text(shown, f, m, x+legendStep, y+legendBox-legendGap, scene.Left, scene.Use(scene.TextColor)))
		}
	}
	return out, longest
}

// value is a section's value as the legend writes it: Go's shortest decimal, which is what
// JavaScript writes but for -0, which is 0 there and here, and the numbers at or past 1e21
// or below 1e-6, and +Inf, which JavaScript writes with an exponent or as Infinity.
func value(v float64) string {
	if v == 0 {
		v = 0
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// fill is the token of the color at place i of the theme's twelve, which scaleOrdinal
// gives the section at place i, the twelve over again past them.
func fill(i int) scene.Token { return scene.PieFill1 + scene.Token(i%12) }

func mark(p scene.Path, box scene.Rect) scene.Item {
	return scene.Item{Kind: scene.Mark, Box: box, Paths: []scene.Path{p}}
}

// text is s set in f on the baseline y, its anchor at x as align says.
func text(s string, f scene.Font, m scene.Measurer, x, y float64, align scene.Align, paint scene.Paint) scene.Item {
	w := m.Measure(s, f).Width
	if align == scene.Center {
		x -= w / 2
	}
	ascent, descent := label.Extent(m, f)
	return scene.Item{Kind: scene.Label, Box: scene.Rect{X: x, Y: y - ascent, W: w, H: ascent + descent},
		Texts: []scene.Text{{Font: f, Lines: []scene.Line{{S: s, X: x, Baseline: y}}, Paint: paint}}}
}
