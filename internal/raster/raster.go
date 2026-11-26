// Package raster draws a scene as an image, in pure Go: its paths with
// rasterx, its text with the outlines of the system's fonts.
package raster

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"unicode"

	"github.com/srwiley/rasterx"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"

	"github.com/bencode/sumi/scene"
)

// Render draws s scale times its size in DIPs, its tokens in p's colors, on
// its background.
func Render(s *scene.Scene, p scene.Palette, scale float64, fonts *Fonts) (*image.RGBA, error) {
	w, h := int(math.Ceil(s.Width*scale)), int(math.Ceil(s.Height*scale))
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if c, ok := p.Resolve(scene.Use(scene.Background)); ok {
		draw.Draw(img, img.Bounds(), image.NewUniform(nrgba(c)), image.Point{}, draw.Src)
	}
	scanner := &scanner{dest: img}
	scanner.Clear()
	r := renderer{scale: scale, fonts: fonts, scanner: scanner, dasher: rasterx.NewDasher(w, h, scanner)}
	for _, it := range s.Items {
		for _, path := range it.Paths {
			r.path(path, p)
		}
		for _, t := range it.Texts {
			if err := r.text(t, p); err != nil {
				return nil, err
			}
		}
	}
	return img, nil
}

type renderer struct {
	scale   float64
	fonts   *Fonts
	scanner *scanner
	dasher  *rasterx.Dasher
	buf     sfnt.Buffer
}

// A scanner is the rasterx.Scanner that fills a path over the box it covers,
// rather than over the whole image as rasterx.ScannerGV does, which for a
// scene of hundreds of paths is the most of the time it takes to draw.
type scanner struct {
	dest     *image.RGBA
	src      image.Uniform
	pts      []fixed.Point26_6
	starts   []bool // whether each point starts a subpath
	min, max fixed.Point26_6
	r        vector.Rasterizer
}

func (s *scanner) Start(a fixed.Point26_6) { s.add(a, true) }
func (s *scanner) Line(b fixed.Point26_6)  { s.add(b, false) }

func (s *scanner) add(p fixed.Point26_6, start bool) {
	s.pts, s.starts = append(s.pts, p), append(s.starts, start)
	s.min = fixed.Point26_6{X: min(s.min.X, p.X), Y: min(s.min.Y, p.Y)}
	s.max = fixed.Point26_6{X: max(s.max.X, p.X), Y: max(s.max.Y, p.Y)}
}

func (s *scanner) Draw() {
	box := image.Rect(s.min.X.Floor(), s.min.Y.Floor(), s.max.X.Ceil()+1, s.max.Y.Ceil()+1).Intersect(s.dest.Bounds())
	if len(s.pts) == 0 || box.Empty() {
		return
	}
	s.r.Reset(box.Dx(), box.Dy())
	for i, p := range s.pts {
		x, y := float32(p.X)/64-float32(box.Min.X), float32(p.Y)/64-float32(box.Min.Y)
		// No ClosePath: rasterx outlines a stroke in open pieces, which close
		// only together, as vector's MoveTo leaves them.
		if s.starts[i] {
			s.r.MoveTo(x, y)
		} else {
			s.r.LineTo(x, y)
		}
	}
	s.r.Draw(s.dest, box, &s.src, image.Point{})
}

func (s *scanner) GetPathExtent() fixed.Rectangle26_6 {
	return fixed.Rectangle26_6{Min: s.min, Max: s.max}
}

func (s *scanner) SetColor(c any) {
	if c, ok := c.(color.Color); ok {
		s.src.C = c
	}
}

func (s *scanner) Clear() {
	s.pts, s.starts = s.pts[:0], s.starts[:0]
	s.min = fixed.Point26_6{X: math.MaxInt32, Y: math.MaxInt32}
	s.max = fixed.Point26_6{X: math.MinInt32, Y: math.MinInt32}
}

func (s *scanner) SetBounds(int, int)           {}
func (s *scanner) SetWinding(bool)              {}
func (s *scanner) SetClip(rect image.Rectangle) {}

func (r *renderer) path(path scene.Path, p scene.Palette) {
	if c, ok := p.Resolve(path.Fill); ok {
		f := &r.dasher.Filler
		r.scanner.SetColor(nrgba(c))
		r.add(f, path.Segs)
		f.Draw()
		f.Clear()
	}
	st := path.Stroke
	if c, ok := p.Resolve(st.Paint); ok && st.Width > 0 {
		dashes := make([]float64, len(st.Dash))
		for i, d := range st.Dash {
			dashes[i] = d * r.scale
		}
		r.dasher.SetStroke(r.fix(st.Width), fixed.I(4), rasterx.RoundCap, nil, rasterx.RoundGap, rasterx.Round, dashes, 0)
		r.scanner.SetColor(nrgba(c))
		r.add(r.dasher, path.Segs)
		r.dasher.Draw()
		r.dasher.Clear()
	}
}

// add adds segs to a, ending each subpath: closed at a Close, open otherwise.
func (r *renderer) add(a rasterx.Rasterx, segs []scene.Seg) {
	open := false
	for _, s := range segs {
		pt := func(i int) fixed.Point26_6 { return r.point(s.Pts[i].X, s.Pts[i].Y) }
		switch s.Op {
		case scene.MoveTo:
			if open {
				a.Stop(false)
			}
			a.Start(pt(0))
			open = true
		case scene.LineTo:
			a.Line(pt(0))
		case scene.QuadTo:
			a.QuadBezier(pt(0), pt(1))
		case scene.CubeTo:
			a.CubeBezier(pt(0), pt(1), pt(2))
		case scene.Close:
			a.Stop(true)
			open = false
		}
	}
	if open {
		a.Stop(false)
	}
}

// text fills the outlines of t's glyphs, each character taken from the first
// font of its chain that has it.
func (r *renderer) text(t scene.Text, p scene.Palette) error {
	c, ok := p.Resolve(t.Paint)
	if !ok {
		return nil
	}
	chain := r.fonts.chain(t.Font)
	ppem := r.fix(t.Font.Size)
	f := &r.dasher.Filler
	r.scanner.SetColor(nrgba(c))
	for _, l := range t.Lines {
		x, y := r.fix(l.X), r.fix(l.Baseline)
		for _, ch := range l.S {
			// Controls and format characters, as a soft hyphen or a zero-width
			// space, show nothing and take no width, as scene.Arial measures them.
			if unicode.In(ch, unicode.Cc, unicode.Cf) {
				continue
			}
			face, g := r.glyph(chain, ch)
			segs, err := face.LoadGlyph(&r.buf, g, ppem, nil)
			if err != nil {
				return err
			}
			addGlyph(f, segs, fixed.Point26_6{X: x, Y: y})
			adv, err := face.GlyphAdvance(&r.buf, g, ppem, font.HintingNone)
			if err != nil {
				return err
			}
			x += adv
		}
	}
	f.Draw()
	f.Clear()
	return nil
}

// glyph is the first font of chain that has ch, and ch's glyph in it; the
// first font's missing glyph when none has it.
func (r *renderer) glyph(chain []*sfnt.Font, ch rune) (*sfnt.Font, sfnt.GlyphIndex) {
	for _, f := range chain {
		if g, err := f.GlyphIndex(&r.buf, ch); err == nil && g != 0 {
			return f, g
		}
	}
	return chain[0], 0
}

// addGlyph adds a glyph's outline, its origin at o.
func addGlyph(f *rasterx.Filler, segs sfnt.Segments, o fixed.Point26_6) {
	open := false
	for _, s := range segs {
		a := func(i int) fixed.Point26_6 { return s.Args[i].Add(o) }
		switch s.Op {
		case sfnt.SegmentOpMoveTo:
			if open {
				f.Stop(true)
			}
			f.Start(a(0))
			open = true
		case sfnt.SegmentOpLineTo:
			f.Line(a(0))
		case sfnt.SegmentOpQuadTo:
			f.QuadBezier(a(0), a(1))
		case sfnt.SegmentOpCubeTo:
			f.CubeBezier(a(0), a(1), a(2))
		}
	}
	if open {
		f.Stop(true)
	}
}

func (r *renderer) point(x, y float64) fixed.Point26_6 {
	return fixed.Point26_6{X: r.fix(x), Y: r.fix(y)}
}

// fix is v DIPs as pixels in 26.6 fixed point.
func (r *renderer) fix(v float64) fixed.Int26_6 { return fixed.Int26_6(math.Round(v * r.scale * 64)) }

func nrgba(c scene.Color) color.NRGBA { return color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A} }
