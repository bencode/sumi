// Package svg writes a scene as SVG, its colors settled by a palette.
package svg

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/bencode/sumi/scene"
)

// Render writes s as an SVG document, painting its tokens in p's colors. Each
// item is a <g> with its kind as class, a path is a <path>, and each line of
// text a <text> set on its baseline.
func Render(s *scene.Scene, p scene.Palette) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%s" height="%s" viewBox="0 0 %s %s">`+"\n",
		num(s.Width), num(s.Height), num(s.Width), num(s.Height))
	for _, it := range s.Items {
		item(&b, it, p)
	}
	b.WriteString("</svg>\n")
	return b.Bytes()
}

func item(b *bytes.Buffer, it scene.Item, p scene.Palette) {
	fmt.Fprintf(b, `<g class="%s"`, it.Kind)
	attrs := [][2]string{{"id", it.ID}, {"data-parent", it.Parent}, {"data-from", it.From}, {"data-to", it.To}}
	for _, a := range attrs {
		if a[1] != "" {
			attr(b, a[0], a[1])
		}
	}
	b.WriteString(">\n")
	for _, path := range it.Paths {
		pathElem(b, path, p)
	}
	for _, t := range it.Texts {
		for _, l := range t.Lines {
			textElem(b, t, l, p)
		}
	}
	b.WriteString("</g>\n")
}

func pathElem(b *bytes.Buffer, path scene.Path, p scene.Palette) {
	b.WriteString(`<path`)
	attr(b, "d", pathData(path.Segs))
	paint(b, "fill", path.Fill, p)
	st := path.Stroke
	if c, ok := p.Resolve(st.Paint); ok && st.Width > 0 {
		color(b, "stroke", c)
		attr(b, "stroke-width", num(st.Width))
		attr(b, "stroke-linecap", "round")
		attr(b, "stroke-linejoin", "round")
		if len(st.Dash) > 0 {
			attr(b, "stroke-dasharray", nums(st.Dash))
		}
	}
	b.WriteString("/>\n")
}

func textElem(b *bytes.Buffer, t scene.Text, l scene.Line, p scene.Palette) {
	b.WriteString(`<text`)
	attr(b, "x", num(l.X))
	attr(b, "y", num(l.Baseline))
	family := "Arial, Helvetica, sans-serif"
	if t.Font.Mono {
		family = "Menlo, Consolas, monospace"
	}
	attr(b, "font-family", family)
	attr(b, "font-size", num(t.Font.Size))
	if t.Font.Bold {
		attr(b, "font-weight", "bold")
	}
	if t.Font.Italic {
		attr(b, "font-style", "italic")
	}
	paint(b, "fill", t.Paint, p)
	b.WriteString(">")
	xml.EscapeText(b, []byte(l.S))
	b.WriteString("</text>\n")
}

// paint writes the attribute name for pt: its color, or "none".
func paint(b *bytes.Buffer, name string, pt scene.Paint, p scene.Palette) {
	c, ok := p.Resolve(pt)
	if !ok {
		attr(b, name, "none")
		return
	}
	color(b, name, c)
}

// color writes c as name, and as name-opacity when it is not opaque.
func color(b *bytes.Buffer, name string, c scene.Color) {
	attr(b, name, fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B))
	if c.A != 255 {
		attr(b, name+"-opacity", num(float64(c.A)/255))
	}
}

func attr(b *bytes.Buffer, name, value string) {
	fmt.Fprintf(b, ` %s="`, name)
	xml.EscapeText(b, []byte(value))
	b.WriteString(`"`)
}

var letters = map[scene.Op]string{
	scene.MoveTo: "M", scene.LineTo: "L", scene.QuadTo: "Q", scene.CubeTo: "C", scene.Close: "Z",
}

// points is how many points each command takes.
var points = map[scene.Op]int{scene.MoveTo: 1, scene.LineTo: 1, scene.QuadTo: 2, scene.CubeTo: 3}

func pathData(segs []scene.Seg) string {
	parts := make([]string, 0, len(segs))
	for _, s := range segs {
		var coords []string
		for _, pt := range s.Pts[:points[s.Op]] {
			coords = append(coords, num(pt.X), num(pt.Y))
		}
		parts = append(parts, letters[s.Op]+strings.Join(coords, " "))
	}
	return strings.Join(parts, " ")
}

// num writes v rounded to two decimals, with no trailing zeros.
func num(v float64) string {
	r := math.Round(v*100) / 100
	if r == 0 {
		r = 0 // no "-0"
	}
	return strconv.FormatFloat(r, 'f', -1, 64)
}

func nums(vs []float64) string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = num(v)
	}
	return strings.Join(out, " ")
}
