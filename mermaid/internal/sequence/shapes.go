package sequence

import (
	"math"

	"github.com/bencode/sumi/scene"
)

// The shapes only sequence diagrams draw, as Mermaid's svgDraw.js draws
// them. A marker's points are in the marker's own units, as Mermaid
// defines it, and placed on the line the way SVG places a marker: its
// reference point at the line's end, its x axis along the line, scaled by
// the line's stroke width unless its units are the user's.

type marker struct {
	pts   []scene.Point
	ref   scene.Point
	scale float64
}

var (
	// insertArrowHead: userSpaceOnUse, so not scaled.
	arrowMarker = marker{[]scene.Point{{X: -1, Y: 0}, {X: 10, Y: 5}, {X: 0, Y: 10}}, scene.Point{X: 7.9, Y: 5}, 1}
	// insertArrowFilledHead, on a line 1.5 wide: the head of -).
	openMarker = marker{[]scene.Point{{X: 18, Y: 7}, {X: 9, Y: 13}, {X: 14, Y: 7}, {X: 9, Y: 1}}, scene.Point{X: 15.5, Y: 7}, 1.5}
	// insertArrowCrossHead, on a line 1.5 wide: two strokes, M 1,2 L 6,7
	// and M 6,2 L 1,7, each 1pt wide.
	crossMarker = marker{[]scene.Point{{X: 1, Y: 2}, {X: 6, Y: 7}, {X: 6, Y: 2}, {X: 1, Y: 7}}, scene.Point{X: 4, Y: 4.5}, 1.5}
)

// halfMarkers are the half heads (insertSolidTopArrowHead and the three
// after it): userSpaceOnUse, so not scaled.
var halfMarkers = map[Head]marker{
	SolidTop:    {[]scene.Point{{X: 0, Y: 0}, {X: 10, Y: 8}, {X: 0, Y: 8}}, scene.Point{X: 7.9, Y: 7.25}, 1},
	SolidBottom: {[]scene.Point{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 0, Y: 8}}, scene.Point{X: 7.9, Y: 0.75}, 1},
	StickTop:    {[]scene.Point{{X: 0, Y: 0}, {X: 7, Y: 7}}, scene.Point{X: 7.5, Y: 7}, 1},
	StickBottom: {[]scene.Point{{X: 0, Y: 7}, {X: 7, Y: 0}}, scene.Point{X: 7.5, Y: 0}, 1},
}

// flipped is the half head on a line's other side. Mermaid draws a head at
// a line's start turned half round, so /|- takes the bottom one to show
// above the line.
var flipped = map[Head]Head{SolidTop: SolidBottom, SolidBottom: SolidTop, StickTop: StickBottom, StickBottom: StickTop}

const crossWidth = 1.5 * 4.0 / 3 // 1pt, scaled with the marker

// at places m's points at p, pointing along dir.
func (m marker) at(p, dir scene.Point) []scene.Point {
	n := math.Hypot(dir.X, dir.Y)
	ux, uy := dir.X/n, dir.Y/n
	out := make([]scene.Point, len(m.pts))
	for i, q := range m.pts {
		rx, ry := (q.X-m.ref.X)*m.scale, (q.Y-m.ref.Y)*m.scale
		out[i] = scene.Point{X: p.X + rx*ux - ry*uy, Y: p.Y + rx*uy + ry*ux}
	}
	return out
}

func polygon(pts []scene.Point) []scene.Seg {
	p := (&scene.Path{}).MoveTo(pts[0].X, pts[0].Y)
	for _, q := range pts[1:] {
		p.LineTo(q.X, q.Y)
	}
	return p.Close().Segs
}

// head is the head at p of a message line arriving along dir.
func head(h Head, p, dir scene.Point) []scene.Path {
	fill := scene.Use(scene.LineColor)
	if h == Arrow {
		return []scene.Path{{Segs: polygon(arrowMarker.at(p, dir)), Fill: fill}}
	}
	if h == Open {
		return []scene.Path{{Segs: polygon(openMarker.at(p, dir)), Fill: fill}}
	}
	if h == Cross {
		q := crossMarker.at(p, dir)
		segs := (&scene.Path{}).MoveTo(q[0].X, q[0].Y).LineTo(q[1].X, q[1].Y).
			MoveTo(q[2].X, q[2].Y).LineTo(q[3].X, q[3].Y).Segs
		return []scene.Path{{Segs: segs, Stroke: scene.Stroke{Paint: fill, Width: crossWidth}}}
	}
	// A half head has no class, so a solid one takes the text color the
	// diagram's root sets. Mermaid strokes a stick one black, whatever the
	// theme; sumi uses the line's color, so it shows on a dark canvas.
	if m, ok := halfMarkers[h]; ok {
		q := m.at(p, dir)
		if stick(h) {
			return []scene.Path{{Segs: line(q[0].X, q[0].Y, q[1].X, q[1].Y), Stroke: scene.Stroke{Paint: fill, Width: 1.5}}}
		}
		return []scene.Path{{Segs: polygon(q), Fill: scene.Use(scene.TextColor)}}
	}
	return nil
}

// tab is the pentagon a block's keyword sits in, its lower right corner
// cut (drawLabel).
func tab(x, y, w, h float64) []scene.Seg {
	const cut = 7
	return polygon([]scene.Point{
		{X: x, Y: y}, {X: x + w, Y: y}, {X: x + w, Y: y + h - cut},
		{X: x + w - cut*1.2, Y: y + h}, {X: x, Y: y + h},
	})
}

func line(x0, y0, x1, y1 float64) []scene.Seg {
	return (&scene.Path{}).MoveTo(x0, y0).LineTo(x1, y1).Segs
}
