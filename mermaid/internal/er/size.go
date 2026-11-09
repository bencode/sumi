package er

import (
	"math"
	"regexp"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/shape"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// erBox's numbers, from Mermaid's default config. The paddings of an entity with
// attributes are 1.25 times the config's: erBox scales them unless config.htmlLabels is
// set, and it is left unset even though labels are HTML.
const (
	rectPadX     = 20  // er.diagramPadding
	rectPadY     = 30  // 1.5 times that
	minWidth     = 100 // er.minEntityWidth
	rectWrap     = 120 // flowchart.wrappingWidth
	estimateSize = 16  // config.fontSize
	padding      = 25.0
	textPadding  = 18.75
)

var lineBreak = regexp.MustCompile(`(?i)</?br\s*/?>`) // lineBreakRegex

// Placed is a text in an entity box.
type Placed struct {
	Box label.Box
	At  scene.Point // the box's top left, from the node's center
}

// A Row is an attribute's row. Keys and Comment have no size when empty.
type Row struct {
	Y, H                      float64 // from the node's center
	Type, Name, Keys, Comment Placed
}

// A Table is what erBox draws inside an entity that has attributes.
type Table struct {
	Name    Placed
	Rows    []Row
	Rule    float64   // the y of the line under the name
	Columns []float64 // the x of the lines between columns
}

// Sized is a node as insertMeasuredNode leaves it. A subgraph's is left empty.
type Sized struct {
	Shape shape.Node
	Label label.Box // of an entity with no attributes
	Table *Table    // of one with attributes
}

// Size sizes the nodes of d, in d.Nodes's order.
func Size(d *Diagram, m scene.Measurer) ([]Sized, error) {
	out := make([]Sized, len(d.Nodes))
	for i, n := range d.Nodes {
		if n.Group {
			continue
		}
		s, err := sizeEntity(n, m)
		if err != nil {
			return nil, err
		}
		out[i] = s
	}
	return out, nil
}

func sizeEntity(n Node, m scene.Measurer) (Sized, error) {
	name := n.Label
	if n.Alias != "" {
		name = n.Alias
	}
	if len(n.Attributes) == 0 {
		return sizeRect(name, n.At, m)
	}
	head, err := setText(m, name, n.At)
	if err != nil {
		return Sized{}, err
	}
	rows := make([][4]label.Box, len(n.Attributes))
	for i, a := range n.Attributes {
		for c, text := range [4]string{a.Type, a.Name, strings.Join(a.Keys, ","), a.Comment} {
			if rows[i][c], err = setText(m, text, n.At); err != nil {
				return Sized{}, err
			}
		}
	}
	return layOut(head, rows), nil
}

// sizeRect is erBox's branch for an entity with no attributes, and drawRect. Its least
// width depends on an estimate of the label as written, at a larger font, so a label
// just past that estimate's limit makes a box narrower than the least.
func sizeRect(name string, at syntax.Pos, m scene.Measurer) (Sized, error) {
	lines, err := readText(name, at)
	if err != nil {
		return Sized{}, err
	}
	b := label.Set(m, lines, rectWrap, 0)
	w := b.W + 2*rectPadX
	if estimate(m, name)+2*rectPadX < minWidth {
		w = math.Max(w, minWidth)
	}
	return Sized{Shape: shape.Box(w, b.H+2*rectPadY), Label: b}, nil
}

// readText reads a text calculateTextWidth has measured, which throws on a line of
// nothing but what SVG collapses: it has no box.
func readText(text string, at syntax.Pos) ([][]label.Span, error) {
	for _, line := range lineBreak.Split(text, -1) {
		if line != "" && strings.Trim(line, " \t\r\n") == "" {
			return nil, syntax.Errorf(at, "the text %q has a blank line", text)
		}
	}
	return label.Read(text, "markdown", at)
}

// estimate is calculateTextWidth: the widest line, rounded.
func estimate(m scene.Measurer, text string) float64 {
	widest := 0.0
	for _, line := range lineBreak.Split(text, -1) {
		widest = math.Max(widest, m.Measure(line, scene.Font{Size: estimateSize}).Width)
	}
	return math.Round(widest)
}

// setText is addText. Generics go in escaped, along with any other < or > in the text,
// and text escaped that way is put back as plain text: one line, with no style. It
// wraps at an estimate at 16px plus 100, which text at 14px does not reach.
func setText(m scene.Measurer, text string, at syntax.Pos) (label.Box, error) {
	if g := label.GenericTypes(text); g != text {
		text = strings.NewReplacer("<", "&lt;", ">", "&gt;").Replace(g)
	}
	lines, err := readText(text, at)
	if err != nil {
		return label.Box{}, err
	}
	if strings.Contains(text, "&lt;") || strings.Contains(text, "&gt;") {
		lines = plainLine(lines)
	}
	return label.Set(m, lines, math.Inf(1), 0), nil
}

func plainLine(lines [][]label.Span) [][]label.Span {
	var b strings.Builder
	for _, line := range lines {
		for _, s := range line {
			b.WriteString(s.Text)
		}
	}
	return [][]label.Span{{{Text: b.String()}}}
}

// columns is the width of the type, name, keys and comment columns. A name wider than
// them all shares the difference out among those that are there.
func columns(name float64, rows [][4]label.Box) [4]float64 {
	var w [4]float64
	for _, r := range rows {
		for c, b := range r {
			w[c] = math.Max(w[c], b.W+padding)
		}
	}
	there := 4.0
	for c := 2; c < 4; c++ {
		if w[c] <= padding {
			w[c] = 0
			there--
		}
	}
	if extra := name + 2*padding - (w[0] + w[1] + w[2] + w[3]); extra > 0 {
		for c := range w {
			if w[c] > 0 {
				w[c] += extra / there
			}
		}
	}
	return w
}

// layOut is erBox's geometry for an entity with attributes.
func layOut(head label.Box, rows [][4]label.Box) Sized {
	cols := columns(head.W, rows)
	starts := [4]float64{0, cols[0], cols[0] + cols[1], cols[0] + cols[1] + cols[2]}
	w := starts[3] + cols[3]

	heights := make([]float64, len(rows))
	h := head.H + textPadding
	for i, r := range rows {
		heights[i] = math.Max(math.Max(r[0].H, r[1].H), math.Max(r[2].H, r[3].H)) + textPadding
		h += heights[i]
	}

	x0, y0 := -w/2, -h/2
	t := Table{
		Name: Placed{head, scene.Point{X: -head.W / 2, Y: y0 + textPadding/2}},
		Rule: y0 + head.H + textPadding,
	}
	for c := 1; c < 4; c++ {
		if cols[c] > 0 {
			t.Columns = append(t.Columns, x0+starts[c])
		}
	}
	y := t.Rule
	for i, r := range rows {
		cell := func(c int) Placed {
			return Placed{r[c], scene.Point{X: x0 + padding/2 + starts[c], Y: y + textPadding/2}}
		}
		t.Rows = append(t.Rows, Row{y, heights[i], cell(0), cell(1), cell(2), cell(3)})
		y += heights[i]
	}
	return Sized{Shape: shape.Box(w, h), Table: &t}
}
