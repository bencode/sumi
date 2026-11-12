package gantt

import (
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// An Axis is the time scale of a diagram, timeScale in ganttRenderer, and the ticks of
// the axis under its tasks.
type Axis struct {
	Ticks      []Tick
	start, end float64 // the domain, in milliseconds; NaN where no task has the time
	width      float64 // the range, from 0
}

// A Tick is a time on the axis, where it falls and its label.
type Tick struct {
	At    time.Time
	X     float64
	Label string
}

// sidePadding is the room left of the axis for the names of sections, and right of it.
const sidePadding = 75

// axisOf is the axis of d in a diagram width wide: from the earliest start of a task to
// the latest end, over the width less the padding of both sides, with its ticks and
// labels as axisBottom draws them. It is an error where Mermaid never finishes the ticks.
func axisOf(d *Diagram, width float64) (Axis, error) {
	a := Axis{start: math.NaN(), end: math.NaN(), width: width - 2*sidePadding}
	zone := time.UTC
	var first, last *Task // the tasks whose times are the ends of the domain
	for i := range d.Tasks {
		t := &d.Tasks[i]
		if t.Start != nil {
			if v := float64(t.Start.UnixMilli()); first == nil || a.start > v {
				a.start, first, zone = v, t, t.Start.Location()
			}
		}
		if t.End != nil {
			if v := float64(t.End.UnixMilli()); last == nil || a.end < v {
				a.end, last, zone = v, t, t.End.Location()
			}
		}
	}
	s := intervalOf(d, a.start, a.end, zone)
	ts, ok := timeTicks(a.start, a.end, s, zone)
	if !ok {
		at := last.At
		if s != nil {
			at = d.TickIntervalAt
		} else if math.Abs(a.start) > math.Abs(a.end) {
			at = first.At
		}
		return Axis{}, syntax.Errorf(at, "Mermaid 12.3.0 never finishes the ticks of this axis: d3 looks for them past the times a Date holds")
	}
	spec := d.AxisFormat
	if spec == "" {
		spec = "%Y-%m-%d"
		if d.DateFormat == "D" {
			spec = "%d"
		}
	}
	for _, t := range ts {
		a.Ticks = append(a.Ticks, Tick{At: time.UnixMilli(int64(t)).In(zone), X: a.x(t), Label: formatTime(spec, t, zone)})
	}
	return a, nil
}

// x is timeScale(t): where t falls between 0 and the width, rounded, the ends swapped for
// a domain that runs back; the middle for a domain of one time; NaN where the domain or t
// is not a time.
func (a Axis) x(t float64) float64 {
	if math.IsNaN(t) {
		return t
	}
	d0, d1, r0, r1 := a.start, a.end, 0.0, a.width
	if d1 < d0 {
		d0, d1, r0, r1 = d1, d0, r1, r0
	}
	u := 0.5
	if b := d1 - d0; b != 0 {
		u = (t - d0) / b
	}
	return jsRound(r0*(1-u) + r1*u)
}

var (
	tickInterval = regexp.MustCompile(`^([1-9][0-9]*)(millisecond|second|minute|hour|day|week|month)$`)
	// unitLengths are dayjs.duration's, a month a twelfth of 365 days.
	unitLengths = map[string]float64{"millisecond": 1, "second": msPerSecond, "minute": msPerMinute, "hour": msPerHour,
		"day": msPerDay, "week": msPerWeek, "month": 365 * msPerDay / 12}
)

// maxTicks is the most ticks Mermaid estimates a tickInterval may give.
const maxTicks = 10000

// intervalOf is the interval of d's tickInterval, nil for d3's own: where it is not
// written as Mermaid takes it, would give more than maxTicks by Mermaid's estimate, which
// is the time between over the interval's length, or every gives none.
func intervalOf(d *Diagram, start, end float64, zone *time.Location) stepper {
	m := tickInterval.FindStringSubmatch(d.TickInterval)
	if m == nil {
		return nil
	}
	// parseInt: past what a float64 holds the number is Infinity, which ParseFloat gives
	// along with its error of range.
	every, _ := strconv.ParseFloat(m[1], 64)
	if start > end || math.Ceil((end-start)/(every*unitLengths[m[2]])) > maxTicks {
		return nil
	}
	return unitEvery(m[2], every, d.Weekday, zone)
}

// unitEvery is the interval of every so many units of a tickInterval, a week starting
// on weekday.
func unitEvery(unit string, every float64, weekday string, zone *time.Location) stepper {
	switch unit {
	case "millisecond":
		return millisecondEvery(every)
	case "second":
		return timeSecond(zone).every(every)
	case "minute":
		return timeMinute(zone).every(every)
	case "hour":
		return timeHour(zone).every(every)
	case "day":
		return timeDay(zone).every(every)
	case "week":
		i := slices.IndexFunc(weekdays[:], func(w string) bool { return strings.EqualFold(w, weekday) })
		return timeWeek(float64(i), zone).every(every)
	}
	return timeMonth(zone).every(every)
}
