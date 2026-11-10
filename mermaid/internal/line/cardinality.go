package line

import "github.com/bencode/sumi/scene"

// A foot is an ER marker at one end of a line, in user space (markerUnits is
// userSpaceOnUse), centered on the line across it.
type foot struct {
	refX   float64
	bars   []float64 // the x of each upright
	circle float64   // the x of its center, 0 for none
	lens   *float64  // the left x of its lens, nil for none
}

func lensAt(x float64) *float64 { return &x }

const (
	barHalf    = 9
	circleSize = 6
	lensWidth  = 36
	lensBulge  = 18 // how far across the line its control points are
	footWidth  = 2  // themeVariables.strokeWidth
)

// cardinalities are markers.js's neo markers, for a line's start, then its end.
var cardinalities = map[string][2]foot{
	"only_one":     {{refX: 0, bars: []float64{9, 15}}, {refX: 18, bars: []float64{3, 9}}},
	"zero_or_one":  {{refX: 0, bars: []float64{9}, circle: 21}, {refX: 30, bars: []float64{21}, circle: 9}},
	"one_or_more":  {{refX: 18, bars: []float64{42}, lens: lensAt(0)}, {refX: 27, bars: []float64{3}, lens: lensAt(9)}},
	"zero_or_more": {{refX: 18, circle: 45.5, lens: lensAt(0)}, {refX: 39, circle: 11, lens: lensAt(21)}},
}

// draw is f, where at places a point by how far along the line and across it. The
// circle's fill is mainBkg, and it hides the line under it.
func (f foot) draw(ink scene.Paint, at func(along, across float64) scene.Point) []scene.Path {
	pen := scene.Stroke{Paint: ink, Width: footWidth}
	place := func(x, across float64) scene.Point { return at(x-f.refX, across) }
	var out []scene.Path
	if f.circle != 0 {
		out = append(out, scene.Path{Segs: scene.Ellipse(place(f.circle, 0), circleSize, circleSize), Fill: scene.Use(scene.NodeFill), Stroke: pen})
	}
	p := &scene.Path{}
	for _, x := range f.bars {
		a, b := place(x, -barHalf), place(x, barHalf)
		p.MoveTo(a.X, a.Y).LineTo(b.X, b.Y)
	}
	if f.lens != nil {
		l, r := place(*f.lens, 0), place(*f.lens+lensWidth, 0)
		up, down := place(*f.lens+lensWidth/2, -lensBulge), place(*f.lens+lensWidth/2, lensBulge)
		p.MoveTo(l.X, l.Y).QuadTo(up.X, up.Y, r.X, r.Y).QuadTo(down.X, down.Y, l.X, l.Y)
	}
	return append(out, scene.Path{Segs: p.Segs, Stroke: pen})
}
