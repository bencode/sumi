package shape

import (
	"math"

	"github.com/bencode/sumi/scene"
)

// arcPoints is how many points stadium.ts sets along each end.
const arcPoints = 50

// pi is Math.PI: a float64, unlike the constant math.Pi, so that expressions
// with it round as JS rounds them.
var pi = math.Pi

// stadium is stadium.ts: a box with half circles for ends, wide enough that the
// label's corners stay inside the polygon roughjs draws through the arcs' points.
func stadium(lw, lh, _ float64) Node {
	const padX, padY = 40, 24
	h := lh + padY
	inscribed := h * math.Cos(pi/(2*(arcPoints-1)))
	capAtLabel := math.Sqrt(math.Max(0, float64(inscribed*inscribed)-float64(lh*lh)))
	w := max(lw+h/4+padX, float64(1.5*h)+(h-inscribed), lw+padX+h-capAtLabel)
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
	from, to := start*pi/180, end*pi/180
	step := (to - from) / (arcPoints - 1)
	pts := make([]scene.Point, arcPoints)
	for i := range pts {
		a := from + float64(float64(i)*step)
		pts[i] = scene.Point{X: -(cx + float64(r*math.Cos(a))), Y: -(cy + float64(r*math.Sin(a)))}
	}
	return pts
}
