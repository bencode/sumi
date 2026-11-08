package flowchart

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// Styles as the neo look writes them out and Chrome reads them: styles2String
// (handDrawnShapeStyles.ts) gives a node's or a subgraph's outline the last value of
// each property its classes and styles set, and insertEdge (edges.js) writes an
// edge's into its path's style attribute, where the last that Chrome reads counts.

// A styling is what an element's styles set of what sumi draws; nil leaves a part to
// the theme.
type styling struct {
	fill, stroke, color *scene.Color
	width               *float64
	dash                []float64 // when dashed: the author's array, empty for none
	dashed              bool
}

// setters read a value of each property sumi draws into a styling, and say whether
// they could.
var setters = map[string]func(*styling, string) bool{
	"fill":   func(s *styling, v string) bool { return readColor(&s.fill, v) },
	"stroke": func(s *styling, v string) bool { return readColor(&s.stroke, v) },
	"color": func(s *styling, v string) bool {
		return strings.EqualFold(v, "none") || readColor(&s.color, v) // color has no none: Chrome drops it
	},
	"stroke-width": func(s *styling, v string) bool {
		w, ok := length(v)
		if ok {
			s.width = &w
		}
		return ok
	},
	"stroke-dasharray": func(s *styling, v string) bool {
		d, ok := dashArray(v)
		if ok {
			s.dash, s.dashed = d, true
		}
		return ok
	},
}

// refused are the properties that change what Chrome draws of an outline or a label
// and that sumi does not draw, with refusedPrefixes; the label's are isLabelStyle's
// but color. The rest of CSS does nothing to an SVG shape, and is left out as Chrome
// leaves it.
var refused = map[string]bool{
	"opacity": true, "fill-opacity": true, "stroke-opacity": true, "fill-rule": true,
	"stroke-linecap": true, "stroke-linejoin": true, "stroke-miterlimit": true, "stroke-dashoffset": true,
	"paint-order": true, "vector-effect": true, "shape-rendering": true, "filter": true,
	"transform": true, "translate": true, "rotate": true, "scale": true,
	"visibility": true, "display": true, "clip-path": true, "mix-blend-mode": true, "isolation": true, "outline": true,
	"x": true, "y": true, "width": true, "height": true, "r": true, "rx": true, "ry": true, "cx": true, "cy": true, "d": true,
	"line-height": true, "letter-spacing": true, "word-spacing": true, "white-space": true,
	"word-wrap": true, "word-break": true, "overflow-wrap": true, "hyphens": true,
}

var refusedPrefixes = []string{"font", "text-", "animation", "mask", "marker"}

// checkStyles is the error for the first of styles, from a style, classDef or
// linkStyle statement at at, that sets what sumi does not draw: a refused property,
// or a value it cannot read of one it draws. It reads every declaration an entry
// can give, an edge's way.
func checkStyles(at syntax.Pos, styles []string) error {
	for _, d := range edgeDeclarations(styles) {
		key, value := d[0], d[1]
		if set := setters[key]; set != nil && !set(&styling{}, value) {
			return syntax.Unsupported(at, fmt.Sprintf("%s:%s in a style", key, value))
		}
		if refused[key] || slices.ContainsFunc(refusedPrefixes, func(p string) bool { return strings.HasPrefix(key, p) }) {
			return syntax.Unsupported(at, key+" in a style")
		}
	}
	return nil
}

// nodeStyling is a node's or a subgraph's styling. styles2Map writes each entry as
// its key and what follows, up to a further colon, and Chrome reads that up to a ;. A
// further colon before the ; leaves a value no statement lets through.
func nodeStyling(compiled, styles []string) styling {
	var ds [][2]string
	for i, e := range slices.Concat(compiled, styles) {
		key, value, ok := strings.Cut(e, ":")
		if !ok {
			continue
		}
		if i < len(compiled) {
			key = strings.ToLower(key) // a class's color reaches the label by the stylesheet too
		}
		value, _, _ = strings.Cut(value, ";")
		ds = append(ds, [2]string{property(key), strings.TrimSpace(value)})
	}
	return styled(ds)
}

// edgeStyling is an edge's styling: its classes' styles, then its own, as its path's
// style attribute has them.
func edgeStyling(e Edge) styling {
	return styled(edgeDeclarations(slices.Concat(e.Compiled, e.Styles)))
}

// edgeDeclarations are the declarations of styles joined by ;, as CSS reads them.
func edgeDeclarations(styles []string) [][2]string {
	var out [][2]string
	for _, e := range styles {
		for _, d := range strings.Split(e, ";") {
			if key, value, ok := strings.Cut(d, ":"); ok {
				out = append(out, [2]string{property(key), strings.TrimSpace(value)})
			}
		}
	}
	return out
}

// property is a declaration's key: CSS takes a property in any case, but isLabelStyle
// knows color only in these letters, and a color it does not know goes on the shape,
// which draws no text.
func property(key string) string {
	k := strings.TrimSpace(key)
	if strings.EqualFold(k, "color") {
		return k
	}
	return strings.ToLower(k)
}

// styled is what ds set, in turn, of what sumi draws: each property's last value
// that reads. Statements refuse values that do not, so none is left out here that
// Chrome would draw.
func styled(ds [][2]string) styling {
	var s styling
	for _, d := range ds {
		if set := setters[d[0]]; set != nil {
			set(&s, d[1])
		}
	}
	return s
}

// markerStroke is what edges.js looks for in a path's style, in just these letters.
var markerStroke = regexp.MustCompile(`stroke:([^;]+)`)

// markerColor is the color an edge's markers take: the first stroke in its path's
// style, or none.
func markerColor(e Edge) *scene.Color {
	m := markerStroke.FindStringSubmatch(strings.Join(slices.Concat(e.Compiled, e.Styles), ";"))
	if m == nil {
		return nil
	}
	var c *scene.Color
	readColor(&c, strings.TrimSpace(m[1]))
	return c
}

// readColor reads v, a CSS color or none, into c.
func readColor(c **scene.Color, v string) bool {
	if strings.EqualFold(v, "none") {
		*c = &scene.Color{}
		return true
	}
	col, ok := syntax.Color(v)
	if ok {
		*c = &col
	}
	return ok
}

// length is a stroke width: a number of pixels, px or none written after it, not
// negative.
func length(v string) (float64, bool) {
	n, err := strconv.ParseFloat(strings.TrimSuffix(strings.ToLower(v), "px"), 64)
	return n, err == nil && n >= 0 && !math.IsInf(n, 0)
}

// dashArray is a stroke-dasharray: none, or lengths apart by spaces or commas.
func dashArray(v string) ([]float64, bool) {
	if strings.EqualFold(v, "none") {
		return []float64{}, true
	}
	fs := strings.FieldsFunc(v, func(r rune) bool { return r == ' ' || r == ',' || r == '\t' })
	out := make([]float64, len(fs))
	for i, f := range fs {
		n, ok := length(f)
		if !ok {
			return nil, false
		}
		out[i] = n
	}
	return out, len(out) > 0
}

// paint is c, or the theme's t when c is nil.
func paint(c *scene.Color, t scene.Token) scene.Paint {
	if c == nil {
		return scene.Use(t)
	}
	return scene.Fixed(*c)
}

// pen is s's stroke over the theme's t, w wide unless s sets a width.
func (s styling) pen(t scene.Token, w float64) scene.Stroke {
	if s.width != nil {
		w = *s.width
	}
	return scene.Stroke{Paint: paint(s.stroke, t), Width: w}
}

// text is the paint of a label on s: its color, or on a fill of its author's that
// hides nothing below it, the black or white that reads on it, or the theme's.
func (s styling) text() scene.Paint {
	switch {
	case s.color != nil:
		return scene.Fixed(*s.color)
	case s.fill != nil && s.fill.A == 255:
		return scene.Fixed(scene.TextOn(*s.fill))
	}
	return scene.Use(scene.TextColor)
}

// outline is a node's or a subgraph's outline, segs, filled and stroked as s says
// over the theme's fill and stroke tokens, w wide; turn is how much of a subpath's
// length Chrome starts its dashes past where segs's starts.
func (s styling) outline(segs []scene.Seg, fill, stroke scene.Token, w, turn float64) []scene.Path {
	st := s.pen(stroke, w)
	if !s.dashed {
		return []scene.Path{{Segs: segs, Fill: paint(s.fill, fill), Stroke: st}}
	}
	return append([]scene.Path{{Segs: segs, Fill: paint(s.fill, fill)}}, dashed(segs, s.dash, st, turn)...)
}
