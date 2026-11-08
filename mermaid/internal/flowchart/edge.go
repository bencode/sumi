package flowchart

import (
	"github.com/bencode/sumi/mermaid/internal/line"
	"github.com/bencode/sumi/mermaid/internal/style"
	"github.com/bencode/sumi/scene"
)

// Edges in the neo look, from edges.js insertEdge.

// widths are the edge-thickness classes' stroke widths; invisible draws none.
var widths = map[string]float64{"thick": 3.5, "invisible": 0}

// hopEnd is what a rewritten line leaves undrawn at its end, and whether it is
// rewritten at all: applyLineJumpsToSvg reads the neo look's array "0 a x y" with a
// regexp, and writes "0 a l-a-y y" over it and over any stroke-dasharray of the
// author's. A dotted line's y is one of its 2s, so that it comes out solid. An array
// of a negative length, or of no 2s, the regexp does not match.
func hopEnd(e Edge, plain, a, b float64) (float64, bool) {
	if e.Stroke == "dotted" {
		return 2, plain-a-b >= 4
	}
	return b, plain-a-b >= 0
}

// strokes are what shows of e's line, segs, styled by s: a fill its author set, under
// the parts that are stroked. Those are all but the gaps under its markers, or the
// dots and solid stretches of a dotted line, or the author's dashes, which leave no
// gaps; none when the line is no width, or is invisible and its author set it no
// stroke. A line with a crossing is hopped, whether or not one was drawn, and plain
// is its length without them.
func strokes(e Edge, segs []scene.Seg, s style.Styling, plain float64, hopped bool) []scene.Path {
	w, ok := widths[e.Stroke]
	if !ok {
		w = 2
	}
	var out []scene.Path
	if s.Fill != nil && s.Fill.A > 0 {
		out = append(out, scene.Path{Segs: segs, Fill: scene.Fixed(*s.Fill)})
	}
	st := s.Pen(scene.LineColor, w)
	if st.Width == 0 || e.Stroke == "invisible" && s.Stroke == nil { // the class sets no stroke
		return out
	}
	a, b := line.Gap[e.StartMarker], line.Gap[e.EndMarker]
	if end, ok := hopEnd(e, plain, a, b); hopped && ok {
		return append(out, scene.Path{Segs: line.Trimmed(segs, a, end, plain), Stroke: st})
	}
	switch {
	case s.Dashed:
		return append(out, line.Dashed(segs, s.Dash, st, 0)...)
	case e.Stroke == "dotted":
		return append(out, line.Dotted(segs, a, b, plain, st)...)
	}
	return append(out, scene.Path{Segs: line.Trimmed(segs, a, b, plain), Stroke: st})
}

// drawEdge is e's item: what shows of its line, and its markers. Its box is the
// whole line's, as getBBox gives a path's. cs are the crossings it hops.
func drawEdge(e Edge, pts []scene.Point, cs []line.Crossing) scene.Item {
	it := scene.Item{Kind: scene.Edge, ID: e.ID, From: e.Start, To: e.End}
	segs := line.Line(pts, e.StartMarker, e.EndMarker, cs)
	if len(segs) == 0 {
		return it
	}
	it.Box = line.Bounds(segs)
	plain := line.ArcLength(line.Line(pts, e.StartMarker, e.EndMarker, nil))
	it.Paths = strokes(e, segs, edgeStyling(e), plain, len(cs) > 0)
	fill := style.Paint(markerColor(e), scene.LineColor)
	if u, ok := line.Heading(segs, true); ok {
		it.Paths = append(it.Paths, line.Marker(e.StartMarker, segs[0].Pts[0], u, true, fill)...)
	}
	if u, ok := line.Heading(segs, false); ok {
		// rounded ends on a line to the last point
		it.Paths = append(it.Paths, line.Marker(e.EndMarker, segs[len(segs)-1].Pts[0], u, false, fill)...)
	}
	return it
}
