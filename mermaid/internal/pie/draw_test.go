package pie

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// What Mermaid 12.3.0 drew in Chrome, in Arial, to the pixel: the drawing's width, then in
// the order drawn the outer circle's center (o), each slice's color and start (s), with +c
// where it goes to the center, the percentages (%), the title (t), and each legend's
// square (k) and label (l).
func TestLayout(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"a slice under 1% is left out of the pie, not the legend, nor its color",
			"pie\n\"A\": 10\n\"B\": 100\n\"C\": 0.1\n\"D\": 50",
			"w524 o225,225 s1@225,40+c s2@296,54+c s4@54,296+c %6%@252,89 %62%@323,323 %31%@110,148 " +
				"k1@441,181 lA@463,195 k2@441,203 lB@463,217 k3@441,225 lC@463,239 k4@441,247 lD@463,261"},
		{"one slice is a whole circle", "pie\n\"ash\": 100",
			"w539 o225,225 s1@225,40 %100%@225,364 k1@441,214 lash@463,228"},
		{"showData writes the values", "pie showData\n\"GitHub\": 100\n\"GitLab\": 50",
			"w607 o225,225 s1@225,40+c s2@65,318+c %67%@345,294 %33%@105,156 " +
				"k1@441,203 lGitHub [100]@463,217 k2@441,225 lGitLab [50]@463,239"},
		{"all zero, no slice", "pie\n\"Dogs\": 0\n\"a:b\": -0",
			"w552 o225,225 k1@441,203 lDogs@463,217 k2@441,225 la:b@463,239"},
	}
	for _, c := range cases {
		body, title := syntax.Prepare(c.src)
		s, err := Layout(body, title, scene.Arial)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got := describe(s); got != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.name, got, c.want)
		}
	}
}

// A title wider than the pie and its legend is as wide as the drawing, the pie in its
// middle: Chrome drew this one 971.49 wide, the title as wide, the pie's center at 485.74.
func TestLayoutWideTitle(t *testing.T) {
	const long = "A title far wider than the pie and its legend together, so the drawing widens to both sides"
	body, title := syntax.Prepare("pie title " + long + "\n\"a\": 1\n\"b\": 2")
	s, err := Layout(body, title, scene.Arial)
	if err != nil {
		t.Fatal(err)
	}
	o := s.Items[0].Box
	w := scene.Arial.Measure(long, scene.Font{Size: titleSize}).Width
	if cx := o.X + o.W/2; s.Width != w || cx != w/2 {
		t.Errorf("width %g, the pie's center at %g; the title %g wide", s.Width, cx, w)
	}
}

func describe(s *scene.Scene) string {
	r := func(v float64) int { return int(math.Round(v)) }
	out := []string{fmt.Sprintf("w%d", r(s.Width))}
	for i, it := range s.Items {
		switch {
		case i == 0:
			out = append(out, fmt.Sprintf("o%d,%d", r(it.Box.X+it.Box.W/2), r(it.Box.Y+it.Box.H/2)))
		case len(it.Texts) > 0:
			l := it.Texts[0].Lines[0]
			tag, x := "l", l.X
			if it.Texts[0].Paint.Token == scene.PieText {
				tag, x = "%", x+it.Box.W/2
			} else if it.Texts[0].Font.Size == titleSize {
				tag, x = "t", x+it.Box.W/2
			}
			out = append(out, fmt.Sprintf("%s%s@%d,%d", tag, l.S, r(x), r(l.Baseline)))
		case it.Paths[0].Stroke.Width == 1:
			out = append(out, fmt.Sprintf("k%d@%d,%d", it.Paths[0].Fill.Token-scene.PieFill1+1, r(it.Box.X), r(it.Box.Y)))
		default:
			p := it.Paths[0]
			c := ""
			for _, seg := range p.Segs {
				if seg.Op == scene.LineTo {
					c = "+c"
				}
			}
			out = append(out, fmt.Sprintf("s%d@%d,%d%s", p.Fill.Token-scene.PieFill1+1, r(p.Segs[0].Pts[0].X), r(p.Segs[0].Pts[0].Y), c))
		}
	}
	return strings.Join(out, " ")
}
