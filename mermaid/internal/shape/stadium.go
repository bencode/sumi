package shape

import (
	"math"

	"github.com/bencode/sumi/scene"
)

// arcPoints is how many points stadium.ts sets along each end.
const arcPoints = 50

// stadium is stadium.ts: a box with half circles for ends, wide enough that the
// label's corners stay inside the polygon roughjs draws through the arcs' points.
func stadium(lw, lh, _ float64) Node {
	const padX, padY = 40, 24
	h := lh + padY
	inscribed := h * math.Cos(math.Pi/(2*(arcPoints-1)))
	capAtLabel := math.Sqrt(math.Max(0, inscribed*inscribed-lh*lh))
	w := max(lw+h/4+padX, 1.5*h+(h-inscribed), lw+padX+h-capAtLabel)
	r := h / 2
	pts := []scene.Point{{X: -w/2 + r, Y: -h / 2}, {X: w/2 - r, Y: -h / 2}}
	pts = append(pts, circlePoints(-w/2+r, 0, r, 90, 270)...)
	pts = append(pts, scene.Point{X: w/2 - r, Y: h / 2})
	pts = append(pts, circlePoints(w/2-r, 0, r, 270, 450)...)
	return polygon(pts, scene.Point{})
}

// circlePoints is generateCirclePoints in util.ts, which gives each point
// negated: arcPoints points about cx, cy from angle start to end, in degrees.
func circlePoints(cx, cy, r, start, end float64) []scene.Point {
	from, to := start*math.Pi/180, end*math.Pi/180
	step := (to - from) / (arcPoints - 1)
	pts := make([]scene.Point, arcPoints)
	for i := range pts {
		a := from + float64(i)*step
		pts[i] = scene.Point{X: -(cx + r*math.Cos(a)), Y: -(cy + r*math.Sin(a))}
	}
	return pts
}
