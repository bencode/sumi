package class

import (
	"math"

	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/scene"
)

// gap is classBox's PADDING and GAP, as class.padding is unset.
const gap = 12

// A piece is a text of a class box: its box, and the number of lines addText counts.
type piece struct {
	box   label.Box
	lines int
}

// An extent is a bounding box. getBBox leaves out what has no size.
type extent struct {
	x0, y0, x1, y1 float64
	some           bool
}

func (e extent) add(o extent) extent {
	if !o.some {
		return e
	}
	if !e.some {
		return o
	}
	return extent{math.Min(e.x0, o.x0), math.Min(e.y0, o.y0), math.Max(e.x1, o.x1), math.Max(e.y1, o.y1), true}
}

func (e extent) moved(x, y float64) extent {
	return extent{e.x0 + x, e.y0 + y, e.x1 + x, e.y1 + y, e.some}
}

func (e extent) w() float64 { return e.x1 - e.x0 }
func (e extent) h() float64 { return e.y1 - e.y0 }

// stacked is a group of texts as addText leaves them, one under the other: the top of
// each, which it raises by half a line, and the group's extent.
func stacked(ps []piece) ([]float64, extent) {
	tops := make([]float64, len(ps))
	var e extent
	offset := 0.0
	for i, p := range ps {
		tops[i] = offset - p.box.H/(2*float64(p.lines))
		if p.box.W != 0 || p.box.H != 0 {
			e = e.add(extent{0, tops[i], p.box.W, tops[i] + p.box.H, true})
		}
		offset += p.box.H
	}
	return tops, e
}

func placed(ps []piece, tops []float64, x, y float64) []Placed {
	out := make([]Placed, len(ps))
	for i, p := range ps {
		out[i] = Placed{p.box, scene.Point{X: x, Y: y + tops[i]}}
	}
	return out
}

// layOut is textHelper's and classBox's geometry, for classBox's default branches: the
// box's insides, and its width and height. The annotation is empty without one.
func layOut(annotation []piece, title piece, members, methods []piece) (ClassBox, float64, float64) {
	aTops, a := stacked(annotation)
	tTops, t := stacked([]piece{title})
	mTops, m := stacked(members)
	nTops, n := stacked(methods)

	// textHelper puts the groups in place, and takes the box of all of them.
	methodsAt := a.h() + t.h() + 4*gap + gap/2
	if m.h() > 0 {
		methodsAt = a.h() + t.h() + 4*gap + m.h()
	}
	all := a.moved(-a.w()/2, 0).
		add(t.moved(-t.w()/2, a.h())).
		add(m.moved(0, a.h()+t.h()+2*gap)).
		add(n.moved(0, methodsAt))

	// classBox makes room for what is missing, and moves the groups to the center. A box
	// with no members has extra room, which it takes from each group's height, even one
	// with nothing in it.
	empty := len(members) == 0 && len(methods) == 0
	w, h, extra := all.w(), all.h(), 0.0
	if empty {
		h += gap
		extra = gap
	} else if len(methods) == 0 {
		h += 2 * gap
	}
	y := -h / 2
	aH, tH, mH := a.h()-extra/2, t.h()-extra/2, m.h()-extra/2
	top := y + gap - extra

	box := ClassBox{
		Title:   placed([]piece{title}, tTops, -t.w()/2, a.h()+top)[0],
		Members: placed(members, mTops, -w/2, a.h()+t.h()+2*gap+top),
		Methods: placed(methods, nTops, -w/2, aH+tH+math.Max(mH, gap/2)+y+5*gap),
		Dividers: [2]float64{
			aH + tH + y + gap,
			aH + tH + mH + y + 3*gap,
		},
	}
	if len(annotation) > 0 {
		box.Annotation = &placed(annotation, aTops, -a.w()/2, top)[0]
	}
	return box, w + 2*gap, h + 2*gap + 2*extra
}
