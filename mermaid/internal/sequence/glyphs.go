package sequence

import (
	"math"

	"github.com/bencode/sumi/scene"
)

// The participants' glyphs, as Mermaid's drawActorType functions draw them
// under its classic look (svgDraw.js). Each gives its paths and its
// label's middle. Fills and outlines are the actor's colors; the lines and
// the circles of the actor-man class are 2 wide, everything else 1.
//
// At the foot some labels sit lower than at the top: drawing the top one
// sets Mermaid's actor.height to the glyph's bounding box (getBBox), plus a
// labelBoxHeight or two, and the foot one centers its label in that.

var (
	glyphFill = scene.Use(scene.NodeFill)
	thin      = scene.Stroke{Paint: scene.Use(scene.NodeStroke), Width: 1}
	thick     = scene.Stroke{Paint: scene.Use(scene.NodeStroke), Width: 2}
)

// box is a participant's rounded box (drawActorTypeParticipant).
func box(g slot) ([]scene.Path, float64) {
	r := scene.Rect{X: g.x, Y: g.y, W: g.w, H: g.h}
	return []scene.Path{{Segs: scene.RoundRect(r, 3), Fill: glyphFill, Stroke: thin}}, g.y + g.h/2
}

// figure is an actor's stick figure: a torso, arms, two legs, and the head
// drawn over them; its label 35 down (drawActorTypeActor).
func figure(g slot) ([]scene.Path, float64) {
	const arm = 18 // ACTOR_TYPE_WIDTH / 2
	c, y := g.c, g.y
	body := (&scene.Path{}).
		MoveTo(c, y+25).LineTo(c, y+45).
		MoveTo(c-arm, y+33).LineTo(c+arm, y+33).
		MoveTo(c-arm, y+60).LineTo(c, y+45).
		MoveTo(c, y+45).LineTo(c+arm-2, y+60).Segs
	return []scene.Path{
		{Segs: body, Stroke: thick},
		{Segs: scene.Ellipse(scene.Point{X: c, Y: y + 10}, 15, 15), Fill: glyphFill, Stroke: thick},
	}, y + 35 + g.h/2
}

// collections is two boxes, the second 6 left and down over the first,
// with the label on it (drawActorTypeCollections).
func collections(g slot) ([]scene.Path, float64) {
	const off = 6
	back := scene.Rect{X: g.x, Y: g.y, W: g.w, H: g.h}
	front := scene.Rect{X: g.x - off, Y: g.y + off, W: g.w, H: g.h}
	return []scene.Path{
		{Segs: scene.RoundRect(back, 0), Fill: glyphFill, Stroke: thin},
		{Segs: scene.RoundRect(front, 0), Fill: glyphFill, Stroke: thin},
	}, front.Y + g.h/2
}

// queue is a cylinder on its side, open to the right: its outline, then
// the rim of its right end (drawActorTypeQueue).
func queue(g slot) ([]scene.Path, float64) {
	ry := g.h / 2
	rx := ry / (2.5 + g.h/50)
	left := scene.Point{X: g.x + rx, Y: g.y + ry}
	right := scene.Point{X: g.x + g.w - rx, Y: g.y + ry}
	body := (&scene.Path{}).MoveTo(left.X, g.y)
	arc(body, left, rx, ry, 270, 90).LineTo(right.X, g.y+g.h)
	arc(body, right, rx, ry, 90, -90).Close()
	rim := (&scene.Path{}).MoveTo(right.X, g.y)
	arc(rim, right, rx, ry, 270, 90)
	return []scene.Path{
		{Segs: body.Segs, Fill: glyphFill, Stroke: thin},
		{Segs: rim.Segs, Fill: glyphFill, Stroke: thin},
	}, g.y + g.h/2
}

// database is a cylinder standing up, a third as wide as the participant
// and as tall as it is wide; the label is below its top (drawActorTypeDatabase).
// Its bounding box is its height, so at the foot the label's middle is 35
// and half of that height + 20 down.
func database(g slot) ([]scene.Path, float64) {
	d := g.w / 3
	rx := d / 2
	ry := rx / (2.5 + d/50)
	top := scene.Point{X: g.c, Y: g.y + 2*ry}
	bottom := scene.Point{X: g.c, Y: g.y + d}
	p := (&scene.Path{}).MoveTo(g.c-rx, top.Y)
	arc(arc(p, top, rx, ry, 180, 0), top, rx, ry, 0, -180).LineTo(g.c-rx, bottom.Y)
	arc(p, bottom, rx, ry, 180, 0).LineTo(g.c+rx, top.Y)
	mid := g.y + 35 + g.h/2
	if g.foot {
		mid = g.y + 35 + (d+20)/2
	}
	return []scene.Path{{Segs: p.Segs, Fill: glyphFill, Stroke: thin}}, mid
}

// controlMarker is the arrowhead on a control's circle: a marker fixed at
// 172.5 degrees, on a line 2 wide, its outline 1.2 wide.
var controlMarker = marker{
	[]scene.Point{{X: 14.4, Y: 5.6}, {X: 7.2, Y: 10.4}, {X: 8.8, Y: 5.6}, {X: 7.2, Y: 0.8}},
	scene.Point{X: 11, Y: 5.8}, 2,
}

// control is a circle with an arrowhead on its top (drawActorTypeControl).
// Its box is the circle's 44 + 2 labelBoxHeights tall at the foot.
func control(g slot) ([]scene.Path, float64) {
	const r = 22
	cy := g.y + 32
	a := 172.5 * math.Pi / 180
	head := controlMarker.at(scene.Point{X: g.c, Y: cy - r}, scene.Point{X: math.Cos(a), Y: math.Sin(a)})
	mid := g.y + r + 12 + g.h/2
	if g.foot {
		mid = g.y + r + 5 + (2*r+40)/2
	}
	return []scene.Path{
		{Segs: scene.Ellipse(scene.Point{X: g.c, Y: cy}, r, r), Fill: glyphFill, Stroke: thick},
		{Segs: polygon(head), Fill: glyphFill, Stroke: scene.Stroke{Paint: thin.Paint, Width: 1.2 * 2}},
	}, mid
}

// entity is a circle on a line, moved down by r/2 - 5 at the top and r at
// the foot (drawActorTypeEntity). Its box is 44 + 20 tall at the foot.
func entity(g slot) ([]scene.Path, float64) {
	const r = 22
	cy, mid := g.y+25+r/2-5, g.y+30+r/2-5+g.h/2
	if g.foot {
		cy, mid = g.y+10+r, g.y+15+r+(2*r+20)/2
	}
	return []scene.Path{
		{Segs: scene.Ellipse(scene.Point{X: g.c, Y: cy}, r, r), Fill: glyphFill, Stroke: thin},
		{Segs: line(g.c-r, cy+r, g.c+r, cy+r), Stroke: thick},
	}, mid
}

// boundary is a circle with a line out of its left side, ended by a short
// upright one, all moved down by r/2 + 10
// (drawActorTypeBoundary). Its box is 44 + 20 tall at the foot.
func boundary(g slot) ([]scene.Path, float64) {
	const r = 22
	cy := g.y + 12 + r/2 + 10
	mid := g.y + 15 + r/2 + 10 + g.h/2
	if g.foot {
		mid = g.y + 15 + r/2 + 10 + (2*r+20)/2
	}
	bar := (&scene.Path{}).
		MoveTo(g.c-2.5*r, cy).LineTo(g.c-15, cy).
		MoveTo(g.c-2.5*r, cy-10).LineTo(g.c-2.5*r, cy+10).Segs
	return []scene.Path{
		{Segs: bar, Stroke: thick},
		{Segs: scene.Ellipse(scene.Point{X: g.c, Y: cy}, r, r), Fill: glyphFill, Stroke: thick},
	}, mid
}

// arc continues p along the ellipse about c from angle a to angle b, in
// degrees, both multiples of 90, which grow clockwise on the screen: 0 is
// right of c and 90 below it. Each quarter is a cubic curve with
// scene.Ellipse's kappa. It returns p.
func arc(p *scene.Path, c scene.Point, rx, ry float64, a, b int) *scene.Path {
	const kappa = 0.5523
	step := 90
	if b < a {
		step = -90
	}
	at := func(deg int) (x, y, dx, dy float64) {
		s, co := math.Sincos(float64(deg) * math.Pi / 180)
		return c.X + rx*co, c.Y + ry*s, -rx * s, ry * co
	}
	k := kappa * float64(step) / 90
	for d := a; d != b; d += step {
		x0, y0, dx0, dy0 := at(d)
		x1, y1, dx1, dy1 := at(d + step)
		p.CubeTo(x0+k*dx0, y0+k*dy0, x1-k*dx1, y1-k*dy1, x1, y1)
	}
	return p
}
