package gantt

import (
	"cmp"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bencode/sumi/mermaid/internal/canvas"
	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// Mermaid's gantt config, as its defaults set it. sidePadding is both leftPadding and
// rightPadding.
const (
	barHeight            = 20
	barGap               = 4
	rowGap               = barHeight + barGap
	topPadding           = 50
	gridLineStartPadding = 35
	fontSize             = 11
	sectionFontSize      = 11
	titleTopMargin       = 25
	titleSize            = 18 // the titleText class
)

// Layout lays out a diagram's body, prepared by syntax.Prepare, width wide, as
// ganttRenderer draws it. title is the front matter's, which a title statement takes the
// place of.
func Layout(body, title string, m scene.Measurer, env Env, width float64) (*scene.Scene, error) {
	d, err := Read(body, env)
	if err != nil {
		return nil, err
	}
	a, err := axisOf(d, width)
	if err != nil {
		return nil, err
	}
	c := chartOf(d, a, m, width)
	excluded, err := excludeDays(c, env.Zone)
	if err != nil {
		return nil, err
	}
	if d.Title != "" {
		title = syntax.Decode(d.Title)
	}
	items := slices.Concat(excluded, grid(c), sectionRows(c), bars(c), taskTexts(c), sectionTitles(c),
		today(c, env.Now), drawTitle(c, title))
	return &scene.Scene{Width: width, Height: c.h, Items: items}, nil
}

// A chart is what each part of a drawing reads: the diagram and its axis, its size, and
// its tasks in the order they are drawn.
type chart struct {
	d     *Diagram
	a     Axis
	m     scene.Measurer
	w, h  float64
	tasks []Task
	rows  int // the tasks that are not verts, a row each
}

func chartOf(d *Diagram, a Axis, m scene.Measurer, w float64) chart {
	rows := 0
	for _, t := range d.Tasks {
		if !t.Vert {
			rows++
		}
	}
	return chart{d: d, a: a, m: m, w: w, h: 2*topPadding + float64(rows*rowGap), tasks: drawOrder(d.Tasks), rows: rows}
}

// drawOrder is the tasks as drawn: by start, as taskCompare sorts them, then the verts
// last. taskCompare takes a task with no start for equal to any other, which orders
// nothing, and V8's sort leaves such a task wherever its steps take it; here it keeps
// its place, which changes only what lies over what, as nothing of it is drawn where
// others are.
func drawOrder(tasks []Task) []Task {
	out := slices.Clone(tasks)
	var at []int
	for i, t := range out {
		if t.Start != nil {
			at = append(at, i)
		}
	}
	timed := make([]Task, len(at))
	for k, i := range at {
		timed[k] = out[i]
	}
	slices.SortStableFunc(timed, func(a, b Task) int { return cmp.Compare(a.Start.UnixMilli(), b.Start.UnixMilli()) })
	for k, i := range at {
		out[i] = timed[k]
	}
	slices.SortStableFunc(out, func(a, b Task) int { return cmp.Compare(btoi(a.Vert), btoi(b.Vert)) })
	return out
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ms is a task's time in milliseconds, NaN where it is not known, as timeScale takes
// undefined.
func ms(t *time.Time) float64 {
	if t == nil {
		return math.NaN()
	}
	return float64(t.UnixMilli())
}

// sectionRows are the backgrounds of the rows, one for each task but the verts, in the
// order drawn: a task's row is its Order, which no other task shares outside the
// compact mode.
func sectionRows(c chart) []scene.Item {
	var items []scene.Item
	for _, t := range c.tasks {
		if !t.Vert {
			r := scene.Rect{X: 0, Y: float64(t.Order*rowGap+topPadding) - 2, W: c.w - sidePadding/2.0, H: rowGap}
			items = append(items, rect(scene.Mark, r, 0, scene.Use(scene.GroupFill), scene.Stroke{})...)
		}
	}
	return items
}

// sectionTitles are the names of the sections, each in the middle of as many rows as it
// has tasks, the first section at the top. But the rows go by the tasks as written, and
// the titles by categoryHeights' keys, which JavaScript lists as Object.keys does: a
// name that is an array index first, by its number. A section named __proto__ has no
// key, so no title.
func sectionTitles(c chart) []scene.Item {
	heights := map[string]int{}
	var names []string
	for _, t := range c.d.Tasks {
		if t.Vert || t.Section == "__proto__" {
			continue
		}
		if heights[t.Section] == 0 {
			names = append(names, t.Section)
		}
		heights[t.Section]++
	}
	slices.SortStableFunc(names, func(a, b string) int {
		i, ia := arrayIndex(a)
		j, ja := arrayIndex(b)
		if ia && ja {
			return cmp.Compare(i, j)
		}
		return cmp.Compare(btoi(ja), btoi(ia))
	})
	f := scene.Font{Size: sectionFontSize}
	ascent, descent := label.Extent(c.m, f)
	var items []scene.Item
	before := 0
	for _, name := range names {
		rows := lineBreak.Split(name, -1)
		for i := range rows {
			rows[i] = syntax.Decode(rows[i])
		}
		// The text is moved up by half its lines but one, each line after the first is an
		// em lower, and alignment-baseline central sets each line's middle there.
		y := float64(heights[name]*rowGap/2+before*rowGap+topPadding) - float64(len(rows)-1)/2*sectionFontSize + (ascent-descent)/2
		var lines []scene.Line
		for i, s := range collapsed(rows) {
			if s != "" {
				lines = append(lines, line(c.m, s, f, 10, scene.Left, y+float64(i)*sectionFontSize))
			}
		}
		if lines != nil {
			items = append(items, textOf(lines, f, c.m))
		}
		before += heights[name]
	}
	return items
}

// lineBreak is lineBreakRegex of Mermaid's common.
var lineBreak = regexp.MustCompile(`(?i)</?br[` + syntax.JSSpaces + `]*/?>`)

// arrayIndex is the number s names when it is an array index, as JavaScript orders the
// keys of an object: an integer below 2³²-1 written as String writes it.
func arrayIndex(s string) (uint64, bool) {
	n, err := strconv.ParseUint(s, 10, 32)
	return n, err == nil && n < 1<<32-1 && strconv.FormatUint(n, 10) == s
}

// today is the line at the time now, from the title's margin to as far from the bottom,
// unless the diagram turns it off. It is drawn wherever the time falls, the canvas or
// not.
func today(c chart, now time.Time) []scene.Item {
	if c.d.TodayMarker == "off" {
		return nil
	}
	x := lacuna(c.a.x(float64(now.UnixMilli())) + sidePadding)
	return []scene.Item{stroked(scene.Point{X: x, Y: titleTopMargin}, scene.Point{X: x, Y: c.h - titleTopMargin},
		scene.Stroke{Paint: scene.Use(scene.NoteStroke), Width: 2})}
}

// drawTitle is the title, centered over the diagram on the title's margin.
func drawTitle(c chart, title string) []scene.Item {
	s := collapsed([]string{title})[0]
	if s == "" {
		return nil
	}
	f := scene.Font{Size: titleSize}
	return []scene.Item{textOf([]scene.Line{line(c.m, s, f, c.w/2, scene.Center, titleTopMargin)}, f, c.m)}
}

// collapsed is the lines of one text as SVG sets them: a tab or a line break is a space,
// a space after a space is dropped, from one line to the next too, and so are those at
// the start and at the end of them all.
func collapsed(lines []string) []string {
	out := make([]string, len(lines))
	space, last := true, -1
	for i, s := range lines {
		var b strings.Builder
		for _, r := range s {
			if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
				if !space {
					b.WriteByte(' ')
				}
				space = true
				continue
			}
			b.WriteRune(r)
			space = false
		}
		if out[i] = b.String(); out[i] != "" {
			last = i
		}
	}
	if last >= 0 {
		out[last] = strings.TrimSuffix(out[last], " ")
	}
	return out
}

// line is s set in f on baseline, its anchor at x as align says.
func line(m scene.Measurer, s string, f scene.Font, x float64, align scene.Align, baseline float64) scene.Line {
	w := m.Measure(s, f).Width
	switch align {
	case scene.Center:
		x -= w / 2
	case scene.Right:
		x -= w
	}
	return scene.Line{S: s, X: x, Baseline: baseline}
}

// textOf is a label of lines in f, its box that of the lines.
func textOf(lines []scene.Line, f scene.Font, m scene.Measurer) scene.Item {
	ascent, descent := label.Extent(m, f)
	var box scene.Rect
	for _, l := range lines {
		box = canvas.Union(box, scene.Rect{X: l.X, Y: l.Baseline - ascent, W: m.Measure(l.S, f).Width, H: ascent + descent})
	}
	return scene.Item{Kind: scene.Label, Box: box, Texts: []scene.Text{{Font: f, Lines: lines, Paint: scene.Use(scene.TextColor)}}}
}

// rect is r with its corners rounded by radius, as a rect element draws it: at 0 for an
// x or a y that is NaN, and nothing where its width or height is not more than 0.
func rect(kind scene.Kind, r scene.Rect, radius float64, fill scene.Paint, stroke scene.Stroke) []scene.Item {
	if !(r.W > 0 && r.H > 0) {
		return nil
	}
	r.X, r.Y = lacuna(r.X), lacuna(r.Y)
	return []scene.Item{{Kind: kind, Box: r, Paths: []scene.Path{{Segs: scene.RoundRect(r, radius), Fill: fill, Stroke: stroke}}}}
}

// stroked is a line from a to b.
func stroked(a, b scene.Point, s scene.Stroke) scene.Item {
	return scene.Item{Kind: scene.Mark, Box: scene.Rect{X: min(a.X, b.X), Y: min(a.Y, b.Y), W: math.Abs(b.X - a.X), H: math.Abs(b.Y - a.Y)},
		Paths: []scene.Path{{Segs: (&scene.Path{}).MoveTo(a.X, a.Y).LineTo(b.X, b.Y).Segs, Stroke: s}}}
}

// lacuna is v as an attribute of SVG takes it: 0, its initial value, for NaN, which is
// no number.
func lacuna(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	return v
}
