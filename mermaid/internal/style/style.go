// Package style reads the styles a diagram's author sets, as the neo look writes them
// out and Chrome reads them: styles2String (handDrawnShapeStyles.ts) gives a node's or a
// group's outline the last value of each property its classes and styles set, and
// insertEdge (edges.js) writes an edge's into its path's style attribute, where the last
// that Chrome reads counts.
package style

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/line"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// A Styling is what an element's styles set of what sumi draws; nil leaves a part to
// the theme.
type Styling struct {
	Fill, Stroke, Color *scene.Color
	Width               *float64
	Dash                []float64 // when Dashed: the author's array, empty for none
	Dashed              bool
}

// setters read a value of each property sumi draws into a styling, and say whether
// they could.
var setters = map[string]func(*Styling, string) bool{
	"fill":   func(s *Styling, v string) bool { return ReadColor(&s.Fill, v) },
	"stroke": func(s *Styling, v string) bool { return ReadColor(&s.Stroke, v) },
	"color": func(s *Styling, v string) bool {
		return strings.EqualFold(v, "none") || ReadColor(&s.Color, v) // color has no none: Chrome drops it
	},
	"stroke-width": func(s *Styling, v string) bool {
		w, ok := length(v)
		if ok {
			s.Width = &w
		}
		return ok
	},
	"stroke-dasharray": func(s *Styling, v string) bool {
		d, ok := dashArray(v)
		if ok {
			s.Dash, s.Dashed = d, true
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

// Check is the error for the first of styles, from a style, classDef or
// linkStyle statement at at, that sets what sumi does not draw: a refused property,
// or a value it cannot read of one it draws. It reads every declaration an entry
// can give, an edge's way.
func Check(at syntax.Pos, styles []string) error {
	for _, d := range Declarations(styles) {
		key, value := d[0], d[1]
		if set := setters[key]; set != nil && !set(&Styling{}, value) {
			return syntax.Unsupported(at, fmt.Sprintf("%s:%s in a style", key, value))
		}
		if refused[key] || slices.ContainsFunc(refusedPrefixes, func(p string) bool { return strings.HasPrefix(key, p) }) {
			return syntax.Unsupported(at, key+" in a style")
		}
	}
	return nil
}

// ForNode is a node's or a subgraph's styling. styles2Map writes each entry as
// its key and what follows, up to a further colon, and Chrome reads that up to a ;. A
// further colon before the ; leaves a value no statement lets through.
func ForNode(compiled, styles []string) Styling {
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
	return Of(ds)
}

// Declarations are the declarations of styles joined by ;, as CSS reads them.
func Declarations(styles []string) [][2]string {
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

// Of is what ds set, in turn, of what sumi draws: each property's last value
// that reads. Statements refuse values that do not, so none is left out here that
// Chrome would draw.
func Of(ds [][2]string) Styling {
	var s Styling
	for _, d := range ds {
		if set := setters[d[0]]; set != nil {
			set(&s, d[1])
		}
	}
	return s
}

// ReadColor reads v, a CSS color or none, into c.
func ReadColor(c **scene.Color, v string) bool {
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

// Paint is c, or the theme's t when c is nil.
func Paint(c *scene.Color, t scene.Token) scene.Paint {
	if c == nil {
		return scene.Use(t)
	}
	return scene.Fixed(*c)
}

// Pen is s's stroke over the theme's t, w wide unless s sets a width.
func (s Styling) Pen(t scene.Token, w float64) scene.Stroke {
	if s.Width != nil {
		w = *s.Width
	}
	return scene.Stroke{Paint: Paint(s.Stroke, t), Width: w}
}

// Text is the paint of a label on s: its color, or on a fill of its author's that
// hides nothing below it, the black or white that reads on it, or the theme's.
func (s Styling) Text() scene.Paint {
	switch {
	case s.Color != nil:
		return scene.Fixed(*s.Color)
	case s.Fill != nil && s.Fill.A == 255:
		return scene.Fixed(scene.TextOn(*s.Fill))
	}
	return scene.Use(scene.TextColor)
}

// Outline is a node's or a subgraph's outline, segs, filled and stroked as s says
// over the theme's fill and stroke tokens, w wide; turn is how much of a subpath's
// length Chrome starts its dashes past where segs's starts.
func (s Styling) Outline(segs []scene.Seg, fill, stroke scene.Token, w, turn float64) []scene.Path {
	st := s.Pen(stroke, w)
	if !s.Dashed {
		return []scene.Path{{Segs: segs, Fill: Paint(s.Fill, fill), Stroke: st}}
	}
	return append([]scene.Path{{Segs: segs, Fill: Paint(s.Fill, fill)}}, line.Dashed(segs, s.Dash, st, turn)...)
}
