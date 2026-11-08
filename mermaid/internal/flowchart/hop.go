package flowchart

import (
	"cmp"
	"math"
	"slices"

	"github.com/bencode/sumi/scene"
)

// Line hops, from rendering-elements/lineJump.ts: where two lines cross, one takes a
// small half circle over the other. ELK's afterPaint rewrites the paths this way, on
// by default, before the viewBox is worked out.

const (
	hopRadius    = 6                  // lineHops.ts's JUMP_RADIUS
	hopClearance = 2                  // CORNER_JUMP_CLEARANCE: the straight run kept beside a corner
	hopMinRadius = 3.6                // MIN_USEFUL_RADIUS_RATIO of hopRadius: no smaller hop is drawn
	crossMargin  = 1e-6               // ENDPOINT_EPSILON: segments that meet within it do not cross
	hopBulge     = 0.5522847498307935 // 4/3·tan(π/8): a cubic's handles for a quarter circle
)

// A crossing is a place where a line hops another: at, t along its segment seg.
type crossing struct {
	seg int
	t   float64
	at  scene.Point
}

// A hop is a crossing there is room to draw: at, d along its segment, r its radius.
type hop struct {
	at   scene.Point
	d, r float64
}

// crossings are the crossings each of lines hops (findEdgeIntersections), of the
// points layout gave them, before the markers move their ends in.
func crossings(lines [][]scene.Point) [][]crossing {
	out := make([][]crossing, len(lines))
	for i, a := range lines {
		for j := i + 1; j < len(lines); j++ {
			onA, onB := crossingsOf(a, lines[j])
			out[i], out[j] = append(out[i], onA...), append(out[j], onB...)
		}
	}
	return out
}

// crossingsOf are the crossings of a and of b, the line after it: where a horizontal
// segment crosses a vertical one the horizontal hops, and otherwise the later line.
// Neither hops where either has left its points to round a corner.
func crossingsOf(a, b []scene.Point) (onA, onB []crossing) {
	for si := 0; si+1 < len(a); si++ {
		for sj := 0; sj+1 < len(b); sj++ {
			ta, tb, at, ok := intersect(a[si], a[si+1], b[sj], b[sj+1])
			if !ok || inCorner(a, si, ta) || inCorner(b, sj, tb) {
				continue
			}
			if horizontal(a[si], a[si+1]) && !horizontal(b[sj], b[sj+1]) {
				onA = append(onA, crossing{si, ta, at})
			} else {
				onB = append(onB, crossing{sj, tb, at})
			}
		}
	}
	return onA, onB
}

// intersect is segmentIntersection: where a1a2 and b1b2 cross, inside both.
func intersect(a1, a2, b1, b2 scene.Point) (ta, tb float64, at scene.Point, ok bool) {
	dxa, dya, dxb, dyb := a2.X-a1.X, a2.Y-a1.Y, b2.X-b1.X, b2.Y-b1.Y
	denom := dxa*dyb - dya*dxb
	if denom == 0 {
		return 0, 0, at, false
	}
	dx, dy := b1.X-a1.X, b1.Y-a1.Y
	ta, tb = (dx*dyb-dy*dxb)/denom, (dx*dya-dy*dxa)/denom
	if ta <= crossMargin || ta >= 1-crossMargin || tb <= crossMargin || tb >= 1-crossMargin {
		return 0, 0, at, false
	}
	return ta, tb, scene.Point{X: a1.X + ta*dxa, Y: a1.Y + ta*dya}, true
}

// horizontal is isHorizontalSeg.
func horizontal(a, b scene.Point) bool {
	return math.Abs(b.X-a.X) >= math.Abs(b.Y-a.Y)
}

// inCorner is crossingSitsInRoundedCorner: whether t along segment seg of pts is where
// the rounded line has left the segment.
func inCorner(pts []scene.Point, seg int, t float64) bool {
	l := math.Hypot(pts[seg+1].X-pts[seg].X, pts[seg+1].Y-pts[seg].Y)
	d := t * l
	if seg > 0 && d < cornerCut(pts[seg-1], pts[seg], pts[seg+1], cornerRadius) {
		return true
	}
	return seg+2 < len(pts) && l-d < cornerCut(pts[seg], pts[seg+1], pts[seg+2], cornerRadius)
}

// fit are the hops rewriteEdgePath draws of cs on segment seg, l long, its corners
// taking front and back of it: each kept clear of the corners, then of its neighbors,
// and none that comes out too small.
func fit(cs []crossing, seg int, l, front, back float64) []hop {
	on := slices.DeleteFunc(slices.Clone(cs), func(c crossing) bool { return c.seg != seg })
	slices.SortStableFunc(on, func(a, b crossing) int { return cmp.Compare(a.t, b.t) })
	var hs []hop
	for _, c := range on {
		d := c.t * l
		if r := min(hopRadius, min(d-front, l-back-d)-hopClearance); r >= hopMinRadius {
			hs = append(hs, hop{c.at, d, r})
		}
	}
	for k := 0; k+1 < len(hs); k++ {
		apart := hs[k+1].d - hs[k].d
		if hs[k].r+hs[k+1].r > apart {
			hs[k].r, hs[k+1].r = min(hs[k].r, apart/2), min(hs[k+1].r, apart/2)
		}
	}
	return slices.DeleteFunc(hs, func(h hop) bool { return h.r < hopMinRadius })
}

// arcOver draws emitJump's arc on p, a line going u: straight to the hop, then half a
// circle, up on a horizontal segment and right on a vertical one, as two cubics.
func arcOver(p *scene.Path, h hop, u scene.Point) {
	along := u.Y
	if horizontal(scene.Point{}, u) {
		along = u.X
	}
	n := scene.Point{X: u.Y, Y: -u.X}
	if along < 0 {
		n = scene.Point{X: -u.Y, Y: u.X}
	}
	off := func(s, t float64) scene.Point {
		return scene.Point{X: h.at.X + s*h.r*u.X + t*h.r*n.X, Y: h.at.Y + s*h.r*u.Y + t*h.r*n.Y}
	}
	start, top, end := off(-1, 0), off(0, 1), off(1, 0)
	c1, c2, c3, c4 := off(-1, hopBulge), off(-hopBulge, 1), off(hopBulge, 1), off(1, hopBulge)
	p.LineTo(start.X, start.Y)
	p.CubeTo(c1.X, c1.Y, c2.X, c2.Y, top.X, top.Y)
	p.CubeTo(c3.X, c3.Y, c4.X, c4.Y, end.X, end.Y)
}
