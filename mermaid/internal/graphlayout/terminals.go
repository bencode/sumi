package graphlayout

import (
	"math"
	"slices"

	"github.com/bencode/sumi/mermaid/internal/elk"
)

const (
	markerRoom     = 6 // what ELK leaves past a marker (TERMINAL_LABEL_MARKER_CLEARANCE, render.ts)
	frameClearance = 2 // what a slid text keeps off a frame (TERMINAL_LABEL_FRAME_CLEARANCE, render.ts)
)

// endRoom is what ELK gets for a text at an end with a marker (terminalElkLabels,
// render.ts).
func endRoom(text *elk.Size, marker string) *elk.Size {
	if text == nil {
		return nil
	}
	pad := 0.0
	if marker != "" && marker != "none" {
		pad = markerRoom
	}
	return &elk.Size{Width: text.Width + 2*pad, Height: text.Height + 2*pad}
}

// placeTexts moves the texts at the edges' ends from where ELK put them to where
// they are drawn, in the order render.ts runs these steps.
func placeTexts(routes []route, frames map[string]frame) []route {
	placed := followEndpoints(routes)
	o := obstaclesOf(placed, frames)
	ends := textEnds(placed)
	for _, move := range []func(edgeLabel, []route, textEnd, obstacles) (elk.Vec, bool){slideOffFrames, putOnSide} {
		for _, e := range ends {
			text := *placed[e.route].texts[e.end]
			if center, ok := move(text, placed, e, o); ok {
				placed[e.route].texts[e.end] = &edgeLabel{center, text.size}
			}
		}
	}
	return placed
}

// followEndpoints moves each text with its end, which clipping and straightening
// slide along the node's side (followMovedEndpoints, render.ts).
func followEndpoints(routes []route) []route {
	moved := slices.Clone(routes)
	for i, r := range moved {
		for end, t := range r.texts {
			if t != nil {
				at := r.points[end*(len(r.points)-1)]
				moved[i].texts[end] = &edgeLabel{elk.Vec{X: t.center.X + (at.X - r.ports[end].X), Y: t.center.Y + (at.Y - r.ports[end].Y)}, t.size}
			}
		}
	}
	return moved
}

// textEnd is a text at an edge's end and the end segment it sits beside
// (placedTerminalLabels, render.ts). end is 0 for the start of the edge, 1 for its
// end.
type textEnd struct {
	route, end, segment int
	from, to            elk.Vec
	length              float64
}

func textEnds(routes []route) []textEnd {
	var all []textEnd
	for i, r := range routes {
		for end, t := range r.texts {
			if t == nil || len(r.points) < 2 {
				continue
			}
			segment := end * (len(r.points) - 2)
			from, to := r.points[segment], r.points[segment+1]
			if length := math.Hypot(to.X-from.X, to.Y-from.Y); length > 0 {
				all = append(all, textEnd{i, end, segment, from, to, length})
			}
		}
	}
	return all
}

// obstacles is what a text must not touch that stays put: nodes and the texts on
// edges, as boxes, and the frames of groups, which are hollow, as segments
// (terminalLabelObstacles, render.ts).
type obstacles struct {
	boxes  []edgeLabel
	frames [][2]elk.Vec
}

func obstaclesOf(routes []route, frames map[string]frame) obstacles {
	var o obstacles
	for _, f := range frames {
		c := f.center()
		if !f.group {
			o.boxes = append(o.boxes, edgeLabel{c, elk.Size{Width: f.drawnWidth, Height: f.height}})
			continue
		}
		x1, x2 := c.X-f.drawnWidth/2, c.X+f.drawnWidth/2
		y1, y2 := c.Y-f.height/2, c.Y+f.height/2
		corners := []elk.Vec{{X: x1, Y: y1}, {X: x2, Y: y1}, {X: x2, Y: y2}, {X: x1, Y: y2}}
		for i, a := range corners {
			o.frames = append(o.frames, [2]elk.Vec{a, corners[(i+1)%4]})
		}
	}
	for _, r := range routes {
		if r.label != nil {
			o.boxes = append(o.boxes, *r.label)
		}
	}
	return o
}

// overlap compares edge with edge, as boxesOverlap does: ELK stacks labels so that
// their boxes touch, and a touch must not round into an overlap.
func overlap(a, b edgeLabel) bool {
	return a.center.X-a.size.Width/2 < b.center.X+b.size.Width/2 && b.center.X-b.size.Width/2 < a.center.X+a.size.Width/2 &&
		a.center.Y-a.size.Height/2 < b.center.Y+b.size.Height/2 && b.center.Y-b.size.Height/2 < a.center.Y+a.size.Height/2
}

// hitsFrame tells whether box, grown by clearance, touches a frame.
func (o obstacles) hitsFrame(box edgeLabel, clearance float64) bool {
	grown := edgeLabel{box.center, elk.Size{Width: box.size.Width + 2*clearance, Height: box.size.Height + 2*clearance}}
	return slices.ContainsFunc(o.frames, func(f [2]elk.Vec) bool { return segmentHitsBox(f[0], f[1], &grown) })
}

// blocked tells whether box, meant for the text e, touches anything but that
// text and the segment it sits beside.
func (o obstacles) blocked(box edgeLabel, routes []route, e textEnd, clearance float64) bool {
	if o.hitsFrame(box, clearance) || slices.ContainsFunc(o.boxes, func(b edgeLabel) bool { return overlap(box, b) }) {
		return true
	}
	for i, r := range routes {
		for s := 0; s+1 < len(r.points); s++ {
			if (i != e.route || s != e.segment) && segmentHitsBox(r.points[s], r.points[s+1], &box) {
				return true
			}
		}
		for end, t := range r.texts {
			if t != nil && (i != e.route || end != e.end) && overlap(box, *t) {
				return true
			}
		}
	}
	return false
}

// slideOffFrames slides a text that lies across a frame along its end segment, by
// the shortest distance, until it clears every frame, and never off that segment
// (slideTerminalLabelsOffFrames, render.ts).
func slideOffFrames(text edgeLabel, routes []route, e textEnd, o obstacles) (elk.Vec, bool) {
	if !o.hitsFrame(text, 0) {
		return elk.Vec{}, false
	}
	far, end := e.to, e.from
	if e.end == 1 {
		far, end = e.from, e.to
	}
	dirX, dirY := (end.X-far.X)/e.length, (end.Y-far.Y)/e.length
	half := (math.Abs(dirX)*text.size.Width + math.Abs(dirY)*text.size.Height) / 2
	for step := 1.0; step <= e.length; step++ {
		for _, shift := range []float64{step, -step} {
			moved := elk.Vec{X: text.center.X + shift*dirX, Y: text.center.Y + shift*dirY}
			along := (moved.X-end.X)*dirX + (moved.Y-end.Y)*dirY
			if along+half <= 0 && along-half >= -e.length &&
				!o.blocked(edgeLabel{moved, text.size}, routes, e, frameClearance) {
				return moved, true
			}
		}
	}
	return elk.Vec{}, false
}

// putOnSide mirrors across its end segment a text on the wrong side of it, onto free
// space only: the text at an edge's start goes on the right of its direction of
// travel, the one at its end on the left, and ELK puts them all on one side
// (putTerminalLabelsOnTheirSide, render.ts).
func putOnSide(text edgeLabel, routes []route, e textEnd, o obstacles) (elk.Vec, bool) {
	normalX, normalY := -(e.to.Y-e.from.Y)/e.length, (e.to.X-e.from.X)/e.length
	offset := (text.center.X-e.from.X)*normalX + (text.center.Y-e.from.Y)*normalY
	if (offset > 0) == (e.end == 0) {
		return elk.Vec{}, false
	}
	mirrored := elk.Vec{X: text.center.X - 2*offset*normalX, Y: text.center.Y - 2*offset*normalY}
	return mirrored, !o.blocked(edgeLabel{mirrored, text.size}, routes, e, 0)
}
