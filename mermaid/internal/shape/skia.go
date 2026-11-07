package shape

import "math"

// Chrome draws an SVG arc as Skia's conics and gives a path's getBBox as Skia's
// tight bounds of them, all in float32. A cylinder's size is that box, so the
// steps are ported here: SkPathBuilder::arcTo, SVG's arc, for arcs without
// rotation, and SkPathPriv::ComputeTightBounds. Products are converted to float32
// one by one, so that Go does not fuse them as Chrome's build does not.

type pt struct{ x, y float32 }

func (p pt) add(q pt) pt { return pt{p.x + q.x, p.y + q.y} }
func (p pt) sub(q pt) pt { return pt{p.x - q.x, p.y - q.y} }

// A conic is SkConic: the curve from p[0] to p[2] pulled toward p[1] by weight w.
type conic struct {
	p [3]pt
	w float32
}

const (
	pi32         float32 = 3.14159265 // SK_ScalarPI
	nearlyZero   float32 = 1.0 / (1 << 12)
	sinCosNearly float32 = 1.0 / (1 << 16)
)

func sin32(r float32) float32  { return float32(math.Sin(float64(r))) }
func cos32(r float32) float32  { return float32(math.Cos(float64(r))) }
func tan32(r float32) float32  { return float32(math.Tan(float64(r))) }
func sqrt32(v float32) float32 { return float32(math.Sqrt(float64(v))) }
func atan32(y, x float32) float32 {
	return float32(math.Atan2(float64(y), float64(x)))
}
func abs32(v float32) float32 { return float32(math.Abs(float64(v))) }

// snap is SkScalarSinSnapToZero's and SkScalarCosSnapToZero's step.
func snap(v float32) float32 {
	if abs32(v) <= sinCosNearly {
		return 0
	}
	return v
}

func integral(v float32) bool { return v == float32(math.Floor(float64(v))) }

// arc is SkPathBuilder::arcTo for an SVG arc of radii rx, ry and no rotation from
// p to q, as conics; sweep is SVG's sweep flag. A degenerate arc is a line, and
// arc gives no conics for it.
func arc(p, q pt, rx, ry float32, large, sweep bool) []conic {
	if rx == 0 || ry == 0 || p == q {
		return nil
	}
	rx, ry = abs32(rx), abs32(ry)
	mid := p.sub(q)
	mid = pt{float32(mid.x * 0.5), float32(mid.y * 0.5)}
	scale := float32(float32(mid.x*mid.x)/float32(rx*rx)) + float32(float32(mid.y*mid.y)/float32(ry*ry))
	if scale > 1 {
		scale = sqrt32(scale)
		rx *= scale
		ry *= scale
	}
	sx, sy := 1/rx, 1/ry
	u0, u1 := pt{float32(p.x * sx), float32(p.y * sy)}, pt{float32(q.x * sx), float32(q.y * sy)}
	delta := u1.sub(u0)
	d := float32(delta.x*delta.x) + float32(delta.y*delta.y)
	factor := sqrt32(max(1/d-0.25, 0))
	if !sweep != large { // SkPathDirection::kCCW is !sweep
		factor = -factor
	}
	delta = pt{float32(delta.x * factor), float32(delta.y * factor)}
	center := u0.add(u1)
	center = pt{float32(center.x * 0.5), float32(center.y * 0.5)}
	center = center.add(pt{-delta.y, delta.x})
	u0, u1 = u0.sub(center), u1.sub(center)
	theta1 := atan32(u0.y, u0.x)
	theta2 := atan32(u1.y, u1.x)
	thetaArc := theta2 - theta1
	if thetaArc < 0 && sweep {
		thetaArc += pi32 * 2
	} else if thetaArc > 0 && !sweep {
		thetaArc -= pi32 * 2
	}
	if abs32(thetaArc) < pi32/(1000*1000) {
		return nil
	}
	segments := int(math.Ceil(float64(abs32(thetaArc / (2 * pi32 / 3)))))
	width := thetaArc / float32(segments)
	t := tan32(0.5 * width)
	if math.IsInf(float64(t), 0) || math.IsNaN(float64(t)) {
		return nil
	}
	w := sqrt32(0.5 + float32(cos32(width)*0.5))
	integers := abs32(pi32/2-abs32(width)) <= nearlyZero &&
		integral(rx) && integral(ry) && integral(q.x) && integral(q.y)
	out := make([]conic, segments)
	start, from := theta1, p
	for i := range out {
		end := start + width
		s, c := snap(sin32(end)), snap(cos32(end))
		e := pt{c, s}.add(center)
		ctrl := e.add(pt{float32(t * s), -float32(t * c)})
		m := [2]pt{{float32(ctrl.x * rx), float32(ctrl.y * ry)}, {float32(e.x * rx), float32(e.y * ry)}}
		if integers {
			for j := range m {
				m[j] = pt{round32(m[j].x), round32(m[j].y)}
			}
		}
		out[i] = conic{[3]pt{from, m[0], m[1]}, w}
		from, start = m[1], end
	}
	out[len(out)-1].p[2] = q // arcTo ends the path at q itself
	return out
}

// round32 is SkScalarRoundToScalar.
func round32(v float32) float32 { return float32(math.Floor(float64(v) + 0.5)) }

// A box is the bounds ComputeTightBounds gathers: left, top, right, bottom.
type box struct{ l, t, r, b float32 }

func newBox(p pt) box { return box{p.x, p.y, p.x, p.y} }

func (b *box) add(p pt) {
	b.l, b.t = min(b.l, p.x), min(b.t, p.y)
	b.r, b.b = max(b.r, p.x), max(b.b, p.y)
}

// addConic adds the points of c that bound it: where its x and its y turn, and
// its end (compute_conic_extremas).
func (b *box) addConic(c conic) {
	for _, axis := range [2]func(pt) float32{func(p pt) float32 { return p.x }, func(p pt) float32 { return p.y }} {
		if t, ok := conicExtremum(axis(c.p[0]), axis(c.p[1]), axis(c.p[2]), c.w); ok {
			b.add(c.at(t))
		}
	}
	b.add(c.p[2])
}

// conicExtremum is conic_find_extrema: the t in (0, 1) where one coordinate of
// a conic turns.
func conicExtremum(p0, p1, p2, w float32) (float32, bool) {
	p20 := p2 - p0
	wp10 := float32(w * (p1 - p0))
	a := float32(w*p20) - p20
	bb := p20 - 2*wp10
	roots := unitQuadRoots(a, bb, wp10)
	if len(roots) == 1 {
		return roots[0], true
	}
	return 0, false
}

// unitQuadRoots is SkFindUnitQuadRoots: the roots of A t² + B t + C in (0, 1).
func unitQuadRoots(a, b, c float32) []float32 {
	if a == 0 {
		return unitDivide(-c, b, nil)
	}
	dr := float64(float64(b)*float64(b)) - float64(4*float64(a)*float64(c))
	if dr < 0 {
		return nil
	}
	r := float32(math.Sqrt(dr))
	if math.IsInf(float64(r), 0) {
		return nil
	}
	q := -(b + r) / 2
	if b < 0 {
		q = -(b - r) / 2
	}
	roots := unitDivide(c, q, unitDivide(q, a, nil))
	if len(roots) == 2 {
		if roots[0] > roots[1] {
			roots[0], roots[1] = roots[1], roots[0]
		} else if roots[0] == roots[1] {
			roots = roots[:1]
		}
	}
	return roots
}

// unitDivide is valid_unit_divide: numer/denom appended when it is in (0, 1).
func unitDivide(numer, denom float32, to []float32) []float32 {
	if numer < 0 {
		numer, denom = -numer, -denom
	}
	if denom == 0 || numer == 0 || numer >= denom {
		return to
	}
	r := numer / denom
	if math.IsNaN(float64(r)) || r == 0 {
		return to
	}
	return append(to, r)
}

// at is SkConic::evalAt through SkConicCoeff.
func (c conic) at(t float32) pt {
	p1w := pt{float32(c.p[1].x * c.w), float32(c.p[1].y * c.w)}
	ax, ay := float32(c.p[2].x-2*p1w.x)+c.p[0].x, float32(c.p[2].y-2*p1w.y)+c.p[0].y
	bx, by := 2*(p1w.x-c.p[0].x), 2*(p1w.y-c.p[0].y)
	db := 2 * (c.w - 1)
	da := 0 - db
	nx := float32(float32(float32(ax*t)+bx)*t) + c.p[0].x
	ny := float32(float32(float32(ay*t)+by)*t) + c.p[0].y
	dz := float32(float32(float32(da*t)+db)*t) + 1
	return pt{nx / dz, ny / dz}
}
