package gantt

import (
	"math"
	"time"

	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// grid is the axis under the tasks, as axisBottom draws it in makeGrid: for each tick a
// line from the axis up to the grid's top and its label under the axis. Its own line
// along the axis has a stroke of width 0, so it draws nothing. The ticks are where the
// time falls, not half a pixel right as d3 draws them on a screen of one device pixel to
// a CSS pixel. d3 writes currentColor on each line, which the stylesheet's gridColor on the
// tick does not reach, and the ticks are 0.8 opaque: both the lines and the labels are Tick.
func grid(c chart) []scene.Item {
	bottom := c.h - topPadding
	f := scene.Font{Size: 10}
	var items []scene.Item
	for _, t := range c.a.Ticks {
		x := sidePadding + t.X
		items = append(items, stroked(scene.Point{X: x, Y: bottom}, scene.Point{X: x, Y: gridLineStartPadding},
			scene.Stroke{Paint: scene.Use(scene.Tick), Width: 1}))
		// The label is 3 under the axis, and dy 1em takes its baseline an em lower.
		if s := label.Collapsed([]string{t.Label})[0]; s != "" {
			items = append(items, textOf([]scene.Line{line(c.m, s, f, x, scene.Center, bottom+3+f.Size)}, f, c.m, scene.Use(scene.Tick)))
		}
	}
	return items
}

// excludeDays are the runs of days the diagram excludes, as drawExcludeDays finds them:
// from the earliest start a day at a time, at its time of day, to the latest end, the
// runs from the start of their first day to the end of their last, across the grid. A
// run still open at the latest end is not drawn. There are none where the diagram
// neither excludes nor includes days, or where the times are more than five years
// apart, as dayjs's diff counts years.
func excludeDays(c chart, zone *time.Location) ([]scene.Item, error) {
	d := c.d
	if len(d.Excludes) == 0 && len(d.Includes) == 0 {
		return nil, nil
	}
	var first, last *Task
	for i := range d.Tasks {
		t := &d.Tasks[i]
		if t.Start != nil && (first == nil || t.Start.Before(*first.Start)) {
			first = t
		}
		if t.End != nil && (last == nil || t.End.After(*last.End)) {
			last = t
		}
	}
	if first == nil || last == nil {
		return nil, nil
	}
	lo, hi := *first.Start, *last.End
	if yearDiff(hi, lo) > 5 {
		return nil, nil
	}
	// The years are 0, however far apart the times, only where a month monthDiff counts by
	// is past the times a Date holds. Mermaid then walks every day between them, which
	// takes it about as long as never finishing.
	if hi.Sub(lo) > 6*366*24*time.Hour {
		return nil, syntax.Unsupported(first.At, "excluded days of tasks this far apart at the edges of the times a Date holds")
	}
	var runs [][2]time.Time
	open := false
	for day, ok := lo, true; ok && !day.After(hi); day, ok = add(day, 1, "d") {
		out, err := d.excluded(day)
		if err != nil {
			return nil, syntax.Errorf(first.At, "%s", err)
		}
		switch {
		case out && open:
			runs[len(runs)-1][1] = day
		case out:
			runs, open = append(runs, [2]time.Time{day, day}), true
		default:
			open = false
		}
	}
	if open {
		runs = runs[:len(runs)-1]
	}
	var items []scene.Item
	for _, r := range runs {
		x0 := c.a.x(dayAt(r[0], zone, 0, 0, 0, 0))
		x1 := c.a.x(dayAt(r[1], zone, 23, 59, 59, 999))
		box := scene.Rect{X: x0 + sidePadding, Y: gridLineStartPadding, W: x1 - x0, H: c.h - topPadding - gridLineStartPadding}
		items = append(items, rect(scene.Mark, box, 0, scene.Use(scene.ExcludeFill), scene.Stroke{})...)
	}
	return items, nil
}

// dayAt is t's day at the time of day given, as setHours sets it, for dayjs's startOf
// and endOf a day: NaN where it is past the times a Date holds.
func dayAt(t time.Time, zone *time.Location, h, mi, s, milli float64) float64 {
	y, m, d, _, _, _, _ := fields(t.In(zone))
	at, ok := local(zone, y, m, d, h, mi, s, milli)
	if !ok {
		return math.NaN()
	}
	return float64(at.UnixMilli())
}

// yearDiff is dayjs's a.diff(b, 'year'): the months from b to a over 12, toward 0.
func yearDiff(a, b time.Time) float64 {
	y := monthDiff(a, b) / 12
	if y < 0 {
		return math.Ceil(y)
	}
	return math.Floor(y)
}

// monthDiff is dayjs's monthDiff(a, b), which is moment's: the whole months from a to b,
// and the fraction of the next, negated. It is 0 where a month it counts by is past the
// times a Date holds, as NaN || 0 is.
func monthDiff(a, b time.Time) float64 {
	if a.Day() < b.Day() {
		return -monthDiff(b, a)
	}
	ay, am, _, _, _, _, _ := fields(a)
	by, bm, _, _, _, _, _ := fields(b)
	whole := 12*(by-ay) + (bm - am)
	anchor, ok := add(a, whole, "M")
	if !ok {
		return 0
	}
	back := float64(b.UnixMilli()) < float64(anchor.UnixMilli())
	step := 1.0
	if back {
		step = -1
	}
	next, ok := add(a, whole+step, "M")
	if !ok {
		return 0
	}
	span := float64(next.UnixMilli() - anchor.UnixMilli())
	if back {
		span = -span
	}
	v := -(whole + float64(b.UnixMilli()-anchor.UnixMilli())/span)
	if math.IsNaN(v) || v == 0 {
		return 0
	}
	return v
}
